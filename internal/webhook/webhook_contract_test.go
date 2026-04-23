package webhook

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWebhookResponseSerializesTopicIDs(t *testing.T) {
	response := WebhookResponse{
		ID:          "wh_123",
		Name:        "hook",
		PayloadType: PayloadTypeBeebuzz,
		TopicIDs:    []string{"top_1", "top_2"},
		IsActive:    true,
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
