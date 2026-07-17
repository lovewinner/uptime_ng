package notifier

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"
)

type FeishuNotifier struct {
	WebhookURL string
	db         *gorm.DB
}

type FeishuCardMessage struct {
	MsgType string     `json:"msg_type"`
	Card    FeishuCard `json:"card"`
}

type FeishuCard struct {
	Header   FeishuCardHeader    `json:"header"`
	Elements []FeishuCardElement `json:"elements"`
}

type FeishuCardHeader struct {
	Title    FeishuCardTitle `json:"title"`
	Template string          `json:"template"`
}

type FeishuCardTitle struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}

type FeishuCardElement struct {
	Tag     string             `json:"tag"`
	Text    *FeishuCardText    `json:"text,omitempty"`
	Fields  []FeishuCardField  `json:"fields,omitempty"`
	Actions []FeishuCardAction `json:"actions,omitempty"`
}

type FeishuCardText struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}

type FeishuCardField struct {
	IsShort bool           `json:"is_short"`
	Text    FeishuCardText `json:"text"`
}

type FeishuCardAction struct {
	Tag   string            `json:"tag"`
	Text  FeishuCardText    `json:"text"`
	URL   string            `json:"url"`
	Type  string            `json:"type"`
	Value map[string]string `json:"value"`
}

func NewFeishuNotifier(webhookURL string, db *gorm.DB) *FeishuNotifier {
	return &FeishuNotifier{
		WebhookURL: webhookURL,
		db:         db,
	}
}

func (n *FeishuNotifier) SendText(content string) error {
	msg := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": content,
		},
	}

	return n.post(msg)
}

func (n *FeishuNotifier) SendCard(monitorName string, monitorType string, status string, statusColor string, msg string, url string) error {
	card := FeishuCardMessage{
		MsgType: "interactive",
		Card: FeishuCard{
			Header: FeishuCardHeader{
				Title: FeishuCardTitle{
					Content: fmt.Sprintf("🔔 监控告警 - %s", monitorName),
					Tag:     "plain_text",
				},
				Template: statusColor,
			},
			Elements: []FeishuCardElement{
				{
					Tag: "div",
					Fields: []FeishuCardField{
						{IsShort: true, Text: FeishuCardText{Content: "**监控类型**\n" + monitorType, Tag: "lark_md"}},
						{IsShort: true, Text: FeishuCardText{Content: "**当前状态**\n" + status, Tag: "lark_md"}},
					},
				},
				{
					Tag: "div",
					Text: &FeishuCardText{
						Content: "**详情**\n" + msg,
						Tag:     "lark_md",
					},
				},
				{
					Tag: "hr",
				},
				{
					Tag: "note",
					Text: &FeishuCardText{
						Content: "来自 uptime_ng 监控系统",
						Tag:     "plain_text",
					},
				},
			},
		},
	}

	if url != "" {
		card.Card.Elements = append(card.Card.Elements[:len(card.Card.Elements)-1],
			FeishuCardElement{
				Tag: "action",
				Actions: []FeishuCardAction{
					{
						Tag:  "button",
						Text: FeishuCardText{Content: "查看详情", Tag: "plain_text"},
						URL:  url,
						Type: "primary",
					},
				},
			},
			card.Card.Elements[len(card.Card.Elements)-1],
		)
	}

	return n.post(card)
}

func (n *FeishuNotifier) post(msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	respBody, err := httpPost(n.WebhookURL, "application/json", body)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("feishu api returned invalid json: %s", string(respBody))
	}

	code, ok := result["code"].(float64)
	if !ok {
		return nil
	}
	if code != 0 {
		return fmt.Errorf("feishu api error: code=%v, msg=%v", code, result["msg"])
	}

	return nil
}

func SendFeishuAlert(webhookURL string, monitorName string, monitorType string, isUp bool, msg string) {
	if webhookURL == "" {
		log.Println("[feishu] webhook URL not configured, skipping alert")
		return
	}

	notifier := NewFeishuNotifier(webhookURL, nil)

	status := "DOWN ⚠️"
	statusColor := "red"
	if isUp {
		status = "UP ✅ 已恢复"
		statusColor = "green"
	}

	if err := notifier.SendCard(monitorName, monitorType, status, statusColor, msg, ""); err != nil {
		log.Printf("[feishu] failed to send card: %v", err)
		if err := notifier.SendText(fmt.Sprintf("🔔 监控告警: %s 状态=%s\n详情: %s", monitorName, status, msg)); err != nil {
			log.Printf("[feishu] failed to send text fallback: %v", err)
		}
	}
}

func GetFeishuMessages() []string {
	return []string{
		"Monitor {{NAME}} status changed to {{STATUS}}",
		"Monitor {{NAME}} is DOWN: {{MSG}}",
		"Monitor {{NAME}} recovered: {{MSG}}",
	}
}

func formatTemplate(tpl string, vars map[string]string) string {
	result := tpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}
