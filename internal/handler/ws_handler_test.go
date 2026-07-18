package handler

import (
	"encoding/json"
	"testing"
)

func TestFormatWSMessageOmitsInternalUserID(t *testing.T) {
	data := formatWSMessage(WSMessage{
		Type:    "heartbeat",
		Payload: map[string]string{"status": "up"},
		UserID:  42,
	})

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["type"] != "heartbeat" {
		t.Fatalf("type=%v want heartbeat", got["type"])
	}
	payload, ok := got["payload"].(map[string]any)
	if !ok || payload["status"] != "up" {
		t.Fatalf("payload=%+v", got["payload"])
	}
	if _, ok := got["UserID"]; ok {
		t.Fatalf("internal UserID leaked: %s", string(data))
	}
	if _, ok := got["user_id"]; ok {
		t.Fatalf("internal user_id leaked: %s", string(data))
	}
}
