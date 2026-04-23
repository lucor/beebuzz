package webhook

import (
	"strings"
	"testing"
	"unicode/utf8"

	"lucor.dev/beebuzz/internal/validator"
)

// FuzzExtractPayloadBeebuzz verifies that extractPayload with the standard payload type
// never panics on arbitrary body bytes.
func FuzzExtractPayloadBeebuzz(f *testing.F) {
	f.Add([]byte(`{"title":"Hello","body":"World"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(``))
	f.Add([]byte(`{"title":"","body":""}`))
	f.Add([]byte(`{"title":"t"}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(`{"title":null,"body":null}`))
	f.Add([]byte(strings.Repeat("a", 64*1024)))
	f.Add([]byte("\x00\x01\x02\xff"))
	f.Add([]byte(`{"title":"` + strings.Repeat("x", 10000) + `","body":"b"}`))
	f.Add([]byte(`{"body":"b"}`))                                 // missing title
	f.Add([]byte(`{"title":" ","body":" "}`))                     // whitespace-only values
	f.Add([]byte(`{"title":1,"body":2}`))                         // numeric field types
	f.Add([]byte(`{"title":"Hello","body":"World","extra":"x"}`)) // extra fields ignored

	svc := &Service{}
	wh := &Webhook{PayloadType: PayloadTypeBeebuzz}

	f.Fuzz(func(t *testing.T, body []byte) {
		// Must never panic.
		title, msg, err := svc.extractPayload(wh, body)

		// Valid extraction requires both title and body to be non-empty.
		if err == nil && (title == "" || msg == "") {
			t.Fatalf("extractPayload returned no error but title=%q body=%q", title, msg)
		}
	})
}

// FuzzExtractPayloadCustom verifies that extractPayload with the custom payload type
// never panics on arbitrary body bytes with gjson paths.
func FuzzExtractPayloadCustom(f *testing.F) {
	f.Add([]byte(`{"data":{"title":"T","message":"M"}}`), "data.title", "data.message")
	f.Add([]byte(`{}`), "title", "body")
	f.Add([]byte(`[]`), "0", "1")
	f.Add([]byte(``), "", "")
	f.Add([]byte(`not json`), "x", "y")
	f.Add([]byte(strings.Repeat("{", 100)), "a", "b")
	f.Add([]byte(`{"a":1}`), strings.Repeat("a.", 100)+"b", "a")
	f.Add([]byte("\x00\xff"), "\x00", "\xff")
	f.Add([]byte(`{"title":"T","body":"B"}`), "title", "body")                         // top-level success
	f.Add([]byte(`["T","B"]`), "0", "1")                                               // array-index success
	f.Add([]byte(`{"title":"T"}`), "title", "body")                                    // one-path-found, one-missing
	f.Add([]byte(`{"data":{"title":1,"message":true}}`), "data.title", "data.message") // non-string gjson results
	f.Add([]byte(`{"title":{"nested":"x"},"body":["a"]}`), "title", "body")            // object/array gjson results

	svc := &Service{}

	f.Fuzz(func(t *testing.T, body []byte, titlePath, bodyPath string) {
		wh := &Webhook{
			PayloadType: PayloadTypeCustom,
			TitlePath:   titlePath,
			BodyPath:    bodyPath,
		}

		// Must never panic.
		_, _, _ = svc.extractPayload(wh, body)
	})
}

// FuzzCreateWebhookRequestValidate verifies that CreateWebhookRequest.Validate never panics.
func FuzzCreateWebhookRequestValidate(f *testing.F) {
	f.Add("webhook", "desc", "beebuzz", "", "")
	f.Add("webhook", "desc", "custom", "data.title", "data.body")
	f.Add("webhook", "desc", "custom", ".data.title", "data.body")
	f.Add("", "", "", "", "")
	f.Add("", "", "invalid_type", "", "")
	f.Add(strings.Repeat("a", 65), strings.Repeat("b", 129), "beebuzz", "", "")
	f.Add("\x00", "\xff", "custom", "\x00", "\xff")
	f.Add("webhook", "desc", "custom", "", "data.body")                                                                          // custom missing title_path
	f.Add("webhook", "desc", "custom", "data.title", "")                                                                         // custom missing body_path
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen), strings.Repeat("b", validator.MaxDescriptionLen), "beebuzz", "", "") // exact boundaries
	f.Add(strings.Repeat("é", validator.MaxDisplayNameLen+1), "desc", "beebuzz", "", "")                                         // multibyte name overflow
	f.Add("webhook", strings.Repeat("é", validator.MaxDescriptionLen+1), "beebuzz", "", "")                                      // multibyte desc overflow
	f.Add("webhook", "desc", "beebuzz", "ignored.title", "ignored.body")                                                         // beebuzz with paths (invalid)

	f.Fuzz(func(t *testing.T, name, description, payloadType, titlePath, bodyPath string) {
		req := CreateWebhookRequest{
			Name:        name,
			Description: description,
			PayloadType: PayloadType(payloadType),
			TitlePath:   titlePath,
			BodyPath:    bodyPath,
			Topics:      []string{"topic1"},
		}

		// Must never panic.
		errs := req.Validate()

		if name == "" && len(errs) == 0 {
			t.Fatal("Validate accepted empty name")
		}

		if payloadType != "beebuzz" && payloadType != "custom" && len(errs) == 0 {
			t.Fatalf("Validate accepted invalid payload_type %q", payloadType)
		}

		if utf8.RuneCountInString(name) > validator.MaxDisplayNameLen && len(errs) == 0 {
			t.Fatalf("Validate accepted name with %d runes (max %d)",
				utf8.RuneCountInString(name), validator.MaxDisplayNameLen)
		}

		if utf8.RuneCountInString(description) > validator.MaxDescriptionLen && len(errs) == 0 {
			t.Fatalf("Validate accepted description with %d runes (max %d)",
				utf8.RuneCountInString(description), validator.MaxDescriptionLen)
		}

		// Custom type requires both paths.
		if payloadType == "custom" && (titlePath == "" || bodyPath == "") && len(errs) == 0 {
			t.Fatal("Validate accepted custom webhook without required paths")
		}

		if payloadType == "custom" && strings.HasPrefix(strings.TrimSpace(titlePath), ".") && len(errs) == 0 {
			t.Fatal("Validate accepted custom webhook title_path with leading dot")
		}

		if payloadType == "custom" && strings.HasPrefix(strings.TrimSpace(bodyPath), ".") && len(errs) == 0 {
			t.Fatal("Validate accepted custom webhook body_path with leading dot")
		}

		if payloadType == "beebuzz" && strings.TrimSpace(titlePath) != "" && len(errs) == 0 {
			t.Fatal("Validate accepted beebuzz webhook with title_path")
		}

		if payloadType == "beebuzz" && strings.TrimSpace(bodyPath) != "" && len(errs) == 0 {
			t.Fatal("Validate accepted beebuzz webhook with body_path")
		}
	})
}

// FuzzUpdateWebhookRequestValidate verifies that UpdateWebhookRequest.Validate never panics
// and enforces required name, valid payload_type, and length limits.
func FuzzUpdateWebhookRequestValidate(f *testing.F) {
	f.Add("webhook", "desc", "beebuzz", "", "")
	f.Add("webhook", "desc", "custom", "data.title", "data.body")
	f.Add("webhook", "desc", "custom", "data.title", ".data.body")
	f.Add("", "", "", "", "")
	f.Add("", "", "invalid_type", "", "")
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen+1), strings.Repeat("b", validator.MaxDescriptionLen+1), "beebuzz", "", "")
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen), strings.Repeat("b", validator.MaxDescriptionLen), "custom", "t", "m")
	f.Add("\x00", "\xff", "custom", "\x00", "\xff")
	f.Add("wh", "", "custom", "", "")
	f.Add("webhook", "desc", "custom", "", "data.body")                                      // custom missing title_path
	f.Add("webhook", "desc", "custom", "data.title", "")                                     // custom missing body_path
	f.Add(strings.Repeat("é", validator.MaxDisplayNameLen), "desc", "beebuzz", "", "")       // multibyte name exact
	f.Add(strings.Repeat("é", validator.MaxDisplayNameLen+1), "desc", "beebuzz", "", "")     // multibyte name overflow
	f.Add("webhook", strings.Repeat("é", validator.MaxDescriptionLen), "custom", "t", "m")   // multibyte desc exact
	f.Add("webhook", strings.Repeat("é", validator.MaxDescriptionLen+1), "custom", "t", "m") // multibyte desc overflow
	f.Add("webhook", "desc", "beebuzz", "ignored.title", "ignored.body")                     // beebuzz with paths (invalid)

	f.Fuzz(func(t *testing.T, name, description, payloadType, titlePath, bodyPath string) {
		req := UpdateWebhookRequest{
			Name:        name,
			Description: description,
			PayloadType: PayloadType(payloadType),
			TitlePath:   titlePath,
			BodyPath:    bodyPath,
			Topics:      []string{"topic1"},
		}

		errs := req.Validate()

		if name == "" && len(errs) == 0 {
			t.Fatal("Validate accepted empty name")
		}

		if payloadType != "beebuzz" && payloadType != "custom" && len(errs) == 0 {
			t.Fatalf("Validate accepted invalid payload_type %q", payloadType)
		}

		if utf8.RuneCountInString(name) > validator.MaxDisplayNameLen && len(errs) == 0 {
			t.Fatalf("Validate accepted name with %d runes (max %d)",
				utf8.RuneCountInString(name), validator.MaxDisplayNameLen)
		}

		if utf8.RuneCountInString(description) > validator.MaxDescriptionLen && len(errs) == 0 {
			t.Fatalf("Validate accepted description with %d runes (max %d)",
				utf8.RuneCountInString(description), validator.MaxDescriptionLen)
		}

		// Custom type requires both paths.
		if payloadType == "custom" && (titlePath == "" || bodyPath == "") && len(errs) == 0 {
			t.Fatal("Validate accepted custom webhook without required paths")
		}

		if payloadType == "custom" && strings.HasPrefix(strings.TrimSpace(titlePath), ".") && len(errs) == 0 {
			t.Fatal("Validate accepted custom webhook title_path with leading dot")
		}

		if payloadType == "custom" && strings.HasPrefix(strings.TrimSpace(bodyPath), ".") && len(errs) == 0 {
			t.Fatal("Validate accepted custom webhook body_path with leading dot")
		}

		if payloadType == "beebuzz" && strings.TrimSpace(titlePath) != "" && len(errs) == 0 {
			t.Fatal("Validate accepted beebuzz webhook with title_path")
		}

		if payloadType == "beebuzz" && strings.TrimSpace(bodyPath) != "" && len(errs) == 0 {
			t.Fatal("Validate accepted beebuzz webhook with body_path")
		}
	})
}
