package device

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDeviceResponseSerializesTopicIDs(t *testing.T) {
	response := DeviceResponse{
		ID:            "dev_123",
		Name:          "phone",
		Description:   "desc",
		IsActive:      true,
		PairingStatus: PairingStatusPaired,
		TopicIDs:      []string{"top_1", "top_2"},
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
