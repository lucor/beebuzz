package token

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAPITokenResponseSerializesTopicIDs(t *testing.T) {
	response := APITokenResponse{
		ID:       "tok_123",
		Name:     "cli",
		LastFour: "abcd",
		TopicIDs: []string{"top_1", "top_2"},
		IsActive: true,
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	body := string(data)
	if !strings.Contains(body, `"topic_ids":["top_1","top_2"]`) {
		t.Fatalf("json body = %s, want topic_ids field", body)
	}
	if strings.Contains(body, `"topics"`) {
		t.Fatalf("json body = %s, got legacy topics field", body)
	}
}
