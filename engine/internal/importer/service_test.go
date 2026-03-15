package importer_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
	"github.com/welife-os/welife-os/engine/internal/importer"
	"github.com/welife-os/welife-os/engine/internal/parser"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/task"
)

type stubParser struct{}

func (stubParser) Parse(_ context.Context, _ io.Reader, _ parser.Options) (*chatir.ChatIR, error) {
	return &chatir.ChatIR{
		Platform:         "test",
		ConversationID:   "conv_import",
		ConversationType: chatir.ConversationPrivate,
		Participants: []chatir.Participant{
			{ID: "self", Name: "Me", IsSelf: true},
		},
		Messages: []chatir.Message{
			{
				ID:        "msg_import_1",
				Timestamp: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
				SenderID:  "self",
				Content:   "hello",
				Type:      chatir.MessageText,
			},
		},
	}, nil
}

func (stubParser) Format() parser.Format {
	return parser.Format("stub_format")
}

func (stubParser) Detect(io.ReadSeeker) bool {
	return true
}

func TestImportBindsReturnedTaskID(t *testing.T) {
	store, err := storage.Open(context.Background(), storage.Config{
		Path: t.TempDir() + "/import_test.db",
		Key:  "test-key",
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	registry := parser.NewRegistry()
	registry.Register(stubParser{})

	taskMgr := task.NewManager(1)
	t.Cleanup(func() { _ = taskMgr.Close() })

	service := importer.NewService(registry, store, taskMgr)
	result, err := service.Import(context.Background(), importer.ImportRequest{
		FileName: "test.stub",
		Format:   parser.Format("stub_format"),
		Data:     bytes.NewReader([]byte("payload")),
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	job, err := store.GetImportJob(context.Background(), result.JobID)
	if err != nil {
		t.Fatalf("get import job: %v", err)
	}
	if job.TaskID != result.TaskID {
		t.Fatalf("expected stored task_id %q, got %q", result.TaskID, job.TaskID)
	}
}
