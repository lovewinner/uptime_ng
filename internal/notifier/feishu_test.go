package notifier

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewFeishuCardMessageAddsActionBeforeNote(t *testing.T) {
	card := newFeishuCardMessage("site", "http", "DOWN", "red", "timeout", "https://example.com")

	if card.MsgType != "interactive" {
		t.Fatalf("msg_type=%s want interactive", card.MsgType)
	}
	if card.Card.Header.Template != "red" {
		t.Fatalf("template=%s want red", card.Card.Header.Template)
	}
	if len(card.Card.Elements) != 5 {
		t.Fatalf("elements=%d want 5", len(card.Card.Elements))
	}

	action := card.Card.Elements[3]
	if action.Tag != "action" || len(action.Actions) != 1 {
		t.Fatalf("action element=%+v", action)
	}
	if action.Actions[0].URL != "https://example.com" {
		t.Fatalf("action url=%s", action.Actions[0].URL)
	}
	if card.Card.Elements[4].Tag != "note" {
		t.Fatalf("last element tag=%s want note", card.Card.Elements[4].Tag)
	}
}

func TestFeishuTextMessageJSONShape(t *testing.T) {
	data, err := json.Marshal(FeishuTextMessage{
		MsgType: "text",
		Content: FeishuTextContent{Text: "hello"},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"msg_type":"text"`) || !strings.Contains(got, `"text":"hello"`) {
		t.Fatalf("json=%s", got)
	}
}

func TestFeishuAPIResponseCodePointer(t *testing.T) {
	var noCode feishuAPIResponse
	if err := json.Unmarshal([]byte(`{"StatusCode":0}`), &noCode); err != nil {
		t.Fatalf("unmarshal no code: %v", err)
	}
	if noCode.Code != nil {
		t.Fatalf("missing code should remain nil")
	}

	var withCode feishuAPIResponse
	if err := json.Unmarshal([]byte(`{"code":9499,"msg":"bad"}`), &withCode); err != nil {
		t.Fatalf("unmarshal code: %v", err)
	}
	if withCode.Code == nil || *withCode.Code != 9499 || withCode.Msg != "bad" {
		t.Fatalf("response=%+v", withCode)
	}
}
