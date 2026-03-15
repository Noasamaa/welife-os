package storage_test

import (
	"context"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

func TestSaveConversationBundleRollsBackOnMessageError(t *testing.T) {
	store, err := storage.Open(context.Background(), storage.Config{
		Path: t.TempDir() + "/bundle_test.db",
		Key:  "test-key",
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	err = store.SaveConversationBundle(context.Background(), storage.Conversation{
		ID:               "conv_bundle",
		Platform:         "test",
		ConversationType: "private",
		MessageCount:     1,
	}, []storage.StoredMessage{
		{
			ID:             "msg_bundle_1",
			ConversationID: "missing_conversation",
			Platform:       "test",
			SenderID:       "u1",
			SenderName:     "Alice",
			Content:        "hello",
			MessageType:    "text",
			Timestamp:      "2026-03-15T10:00:00Z",
		},
	}, nil)
	if err == nil {
		t.Fatal("expected bundle save to fail")
	}

	if _, err := store.GetConversation(context.Background(), "conv_bundle"); err == nil {
		t.Fatal("conversation should have been rolled back")
	}
}
