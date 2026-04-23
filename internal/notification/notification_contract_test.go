package notification

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNotificationPayloadSerializesTopicID(t *testing.T) {
	payload := NotificationPayload{
		ID:      "ntf_123",
		Title:   "Alert",
		Body:    "Something happened",
		TopicID: "top_1",
		Topic:   "alerts",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	body := string(data)
	if !strings.Contains(body, `"topic_id":"top_1"`) {
		t.Fatalf("json body = %s, want topic_id field", body)
	}
	if strings.Contains(body, `"topicId"`) {
		t.Fatalf("json body = %s, got camelCase topicId field", body)
	}
}
