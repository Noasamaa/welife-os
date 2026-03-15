package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/welife-os/welife-os/engine/internal/graph"
	"github.com/welife-os/welife-os/engine/internal/importer"
	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/parser"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/task"
)

const Version = "0.1.0"

type Config struct {
	Host          string
	Port          int
	DatabasePath  string
	DatabaseKey   string
	OllamaBaseURL string
	OllamaModel   string
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type Server struct {
	config      Config
	store       *storage.Store
	llmClient   *llm.Client
	taskManager *task.Manager
	importer    *importer.Service
	graphEngine *graph.Engine
	router      http.Handler
	httpServer  *http.Server
	shutdown    sync.Once
	shutdownErr error
}

func New(cfg Config) (*Server, error) {
	store, err := storage.Open(context.Background(), storage.Config{
		Path: cfg.DatabasePath,
		Key:  cfg.DatabaseKey,
	})
	if err != nil {
		return nil, err
	}

	llmClient, err := llm.New(llm.Config{
		BaseURL: cfg.OllamaBaseURL,
		Model:   cfg.OllamaModel,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		_ = store.Close()
		return nil, err
	}

	server := &Server{
		config:      cfg,
		store:       store,
		llmClient:   llmClient,
		taskManager: task.NewManager(2),
	}

	// Initialize parser registry with all built-in parsers
	registry := parser.NewRegistry()
	registry.Register(parser.NewWeChatCSVParser())
	registry.Register(parser.NewTelegramParser())
	registry.Register(parser.NewWhatsAppParser())
	registry.Register(parser.NewGenericCSVParser())

	server.importer = importer.NewService(registry, store, server.taskManager)

	// Initialize graph engine
	extractor := graph.NewExtractor(llmClient)
	server.graphEngine = graph.NewEngine(store, extractor, server.taskManager)
	server.router = server.routes()
	server.httpServer = &http.Server{
		Addr:              cfg.Addr(),
		Handler:           server.router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server, nil
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) ListenAndServe() error {
	err := s.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.shutdown.Do(func() {
		var errs []error
		if s.httpServer != nil {
			if err := s.httpServer.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errs = append(errs, err)
			}
		}
		if s.taskManager != nil {
			if err := s.taskManager.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		if s.store != nil {
			if err := s.store.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		s.shutdownErr = errors.Join(errs...)
	})
	return s.shutdownErr
}
