package chatir

import (
	"encoding/json"
	"testing"
	"time"
)

func TestChatIRValidateNormalizesMessageCount(t *testing.T) {
	chat := &ChatIR{
		Platform:         "wechat",
		ConversationID:   "conv_001",
		ConversationType: ConversationPrivate,
		Participants: []Participant{
			{ID: "user_001", Name: "我", IsSelf: true},
		},
		Messages: []Message{
			{
				ID:        "msg_001",
				Timestamp: time.Now(),
				SenderID:  "user_001",
				Content:   "你好",
				Type:      MessageText,
			},
		},
	}

	if err := chat.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	if chat.Metadata.MessageCount != 1 {
		t.Fatalf("unexpected message count: %d", chat.Metadata.MessageCount)
	}
}

func TestChatIRJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	source := ChatIR{
		Platform:         "telegram",
		ConversationID:   "conv_002",
		ConversationType: ConversationGroup,
		Participants: []Participant{
			{ID: "user_001", Name: "我", IsSelf: true},
			{ID: "user_002", Name: "老王"},
		},
		Messages: []Message{
			{
				ID:        "msg_001",
				Timestamp: now,
				SenderID:  "user_002",
				Content:   "测试",
				Type:      MessageText,
			},
		},
		Metadata: Metadata{
			ExportedAt:   now,
			MessageCount: 1,
			DateRange:    [2]string{"2026-03-01", "2026-03-15"},
		},
	}

	payload, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var target ChatIR
	if err := json.Unmarshal(payload, &target); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if target.ConversationID != source.ConversationID {
		t.Fatalf("conversation id mismatch: %s", target.ConversationID)
	}
	if target.Messages[0].Timestamp.IsZero() {
		t.Fatal("timestamp should not be zero")
	}
}
