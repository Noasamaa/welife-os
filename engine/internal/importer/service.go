package importer

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
	"github.com/welife-os/welife-os/engine/internal/parser"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/task"
)

// Service orchestrates the import pipeline: detect → parse → store.
type Service struct {
	registry *parser.Registry
	store    *storage.Store
	tasks    *task.Manager
}

// NewService creates a new import service.
func NewService(registry *parser.Registry, store *storage.Store, tasks *task.Manager) *Service {
	return &Service{
		registry: registry,
		store:    store,
		tasks:    tasks,
	}
}

// ImportRequest holds parameters for a single import operation.
type ImportRequest struct {
	FileName string
	Format   parser.Format // empty = auto-detect
	Data     io.ReadSeeker
	Options  parser.Options
}

// ImportResult is returned after an import completes.
type ImportResult struct {
	JobID          string `json:"job_id"`
	TaskID         string `json:"task_id"`
	ConversationID string `json:"conversation_id,omitempty"`
	MessageCount   int    `json:"message_count"`
}

// Import starts an async import pipeline and returns immediately with a job ID.
func (s *Service) Import(ctx context.Context, req ImportRequest) (ImportResult, error) {
	// Resolve parser
	var p parser.Parser
	if req.Format != "" {
		var ok bool
		p, ok = s.registry.Get(req.Format)
		if !ok {
			return ImportResult{}, fmt.Errorf("unknown format: %s", req.Format)
		}
	} else {
		var ok bool
		p, ok = s.registry.Detect(req.Data)
		if !ok {
			return ImportResult{}, fmt.Errorf("cannot auto-detect format for %s", req.FileName)
		}
		// Reset after detection
		if _, err := req.Data.Seek(0, io.SeekStart); err != nil {
			return ImportResult{}, err
		}
	}

	// Read all data into memory so the async task owns it
	raw, err := io.ReadAll(req.Data)
	if err != nil {
		return ImportResult{}, fmt.Errorf("reading file data: %w", err)
	}

	jobID := fmt.Sprintf("import_%d", time.Now().UnixNano())

	// Create import job record
	if err := s.store.CreateImportJob(ctx, storage.ImportJob{
		ID:       jobID,
		TaskID:   "", // will be updated after submit
		FileName: req.FileName,
		Format:   string(p.Format()),
		Status:   "pending",
	}); err != nil {
		return ImportResult{}, fmt.Errorf("creating import job: %w", err)
	}

	opts := req.Options
	detectedFormat := p.Format()

	// Submit async task
	taskID := s.tasks.Submit("import:"+req.FileName, func(taskCtx context.Context) error {
		return s.runImport(taskCtx, jobID, raw, p, detectedFormat, opts)
	})

	// Update job with task ID
	_ = s.store.UpdateImportJob(ctx, jobID, "running", "", 0, "")

	return ImportResult{
		JobID:  jobID,
		TaskID: taskID,
	}, nil
}

// runImport executes the import pipeline inside a worker goroutine.
func (s *Service) runImport(ctx context.Context, jobID string, data []byte, p parser.Parser, format parser.Format, opts parser.Options) error {
	// Parse
	ir, err := p.Parse(ctx, bytesReader(data), opts)
	if err != nil {
		_ = s.store.UpdateImportJob(ctx, jobID, "failed", "", 0, err.Error())
		return fmt.Errorf("parsing: %w", err)
	}

	// Store conversation
	conv := storage.Conversation{
		ID:               ir.ConversationID,
		Platform:         ir.Platform,
		ConversationType: string(ir.ConversationType),
		MessageCount:     len(ir.Messages),
	}
	if len(ir.Messages) > 0 {
		conv.FirstMessageAt = ir.Messages[0].Timestamp.Format(time.RFC3339)
		conv.LastMessageAt = ir.Messages[len(ir.Messages)-1].Timestamp.Format(time.RFC3339)
	}
	if err := s.store.SaveConversation(ctx, conv); err != nil {
		_ = s.store.UpdateImportJob(ctx, jobID, "failed", "", 0, err.Error())
		return fmt.Errorf("saving conversation: %w", err)
	}

	// Store messages
	msgs := chatIRToStoredMessages(ir)
	if err := s.store.SaveMessages(ctx, msgs); err != nil {
		_ = s.store.UpdateImportJob(ctx, jobID, "failed", ir.ConversationID, 0, err.Error())
		return fmt.Errorf("saving messages: %w", err)
	}

	// Store participants
	parts := chatIRToStoredParticipants(ir)
	if err := s.store.SaveParticipants(ctx, parts); err != nil {
		_ = s.store.UpdateImportJob(ctx, jobID, "failed", ir.ConversationID, len(msgs), err.Error())
		return fmt.Errorf("saving participants: %w", err)
	}

	// Mark success
	_ = s.store.UpdateImportJob(ctx, jobID, "succeeded", ir.ConversationID, len(msgs), "")
	return nil
}

// chatIRToStoredMessages converts ChatIR messages to storage format.
func chatIRToStoredMessages(ir *chatir.ChatIR) []storage.StoredMessage {
	msgs := make([]storage.StoredMessage, len(ir.Messages))
	// Build sender name lookup
	nameByID := make(map[string]string)
	for _, p := range ir.Participants {
		nameByID[p.ID] = p.Name
	}
	for i, m := range ir.Messages {
		name := nameByID[m.SenderID]
		if name == "" {
			name = m.SenderID
		}
		msgs[i] = storage.StoredMessage{
			ID:             m.ID,
			ConversationID: ir.ConversationID,
			Platform:       ir.Platform,
			SenderID:       m.SenderID,
			SenderName:     name,
			Content:        m.Content,
			MessageType:    string(m.Type),
			ReplyTo:        m.ReplyTo,
			Timestamp:      m.Timestamp.Format(time.RFC3339),
		}
	}
	return msgs
}

// chatIRToStoredParticipants converts ChatIR participants to storage format.
func chatIRToStoredParticipants(ir *chatir.ChatIR) []storage.StoredParticipant {
	parts := make([]storage.StoredParticipant, len(ir.Participants))
	for i, p := range ir.Participants {
		parts[i] = storage.StoredParticipant{
			ConversationID: ir.ConversationID,
			ParticipantID:  p.ID,
			DisplayName:    p.Name,
			IsSelf:         p.IsSelf,
		}
	}
	return parts
}

type bytesReaderWrapper struct {
	*io.SectionReader
}

func bytesReader(data []byte) io.Reader {
	return &simpleReader{data: data, pos: 0}
}

type simpleReader struct {
	data []byte
	pos  int
}

func (r *simpleReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
