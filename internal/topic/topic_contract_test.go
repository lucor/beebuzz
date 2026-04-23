package topic

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTopicResponseDoesNotSerializeUserID(t *testing.T) {
	response := TopicResponse{
		ID:   "top_123",
		Name: "alerts",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	body := string(data)
	if strings.Contains(body, `"user_id"`) {
		t.Fatalf("json body = %s, got redundant user_id field", body)
	}
}
