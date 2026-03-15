package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/welife-os/welife-os/engine/internal/agent"
	"github.com/welife-os/welife-os/engine/internal/forum"
	"github.com/welife-os/welife-os/engine/internal/graph"
	"github.com/welife-os/welife-os/engine/internal/importer"
	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/parser"
	"github.com/welife-os/welife-os/engine/internal/reminder"
	"github.com/welife-os/welife-os/engine/internal/report"
	"github.com/welife-os/welife-os/engine/internal/simulation"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/task"
)

const Version = "1.0.0"

type Config struct {
	Host           string
	Port           int
	DatabasePath   string
	DatabaseKey    string
	LLMProvider    string
	LLMBaseURL     string
	LLMModel       string
	LLMAPIKey      string
	EmbeddingModel string
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type Server struct {
	config      Config
	store       *storage.Store
	llmClient   *llm.SwappableClient
	taskManager *task.Manager
	importer    *importer.Service
	graphEngine *graph.Engine
	forumEngine *forum.Engine
	reportGenerator *report.Generator
	renderer        *report.Renderer
	coachAgent      *agent.CoachAgent
	simEngine       *simulation.Engine
	reminderService *reminder.Service
	vectorStore     storage.VectorStore
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

	// Build initial LLM config from env, then override with DB settings if present.
	llmCfg := llm.Config{
		Provider:       cfg.LLMProvider,
		BaseURL:        cfg.LLMBaseURL,
		Model:          cfg.LLMModel,
		EmbeddingModel: cfg.EmbeddingModel,
		Timeout:        120 * time.Second,
		APIKey:         cfg.LLMAPIKey,
	}
	llmCfg = overrideLLMConfigFromDB(store, llmCfg)

	rawClient, err := llm.NewClient(llmCfg)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	llmClient := llm.NewSwappable(rawClient)

	server := &Server{
		config:      cfg,
		store:       store,
		llmClient:   llmClient,
		taskManager: task.NewManager(2),
		vectorStore: storage.NewSqliteVecStore(store.DB()),
	}

	// Initialize parser registry with all built-in parsers
	registry := parser.NewRegistry()
	registry.Register(parser.NewWeChatCSVParser())
	registry.Register(parser.NewTelegramParser())
	registry.Register(parser.NewWhatsAppParser())
	registry.Register(parser.NewGenericCSVParser())
	registry.Register(parser.NewDiscordParser())
	registry.Register(parser.NewQQParser())
	registry.Register(parser.NewLarkParser())
	registry.Register(parser.NewIMessageParser())

	server.importer = importer.NewService(registry, store, server.taskManager)

	// Initialize graph engine
	extractor := graph.NewExtractor(llmClient)
	server.graphEngine = graph.NewEngine(store, extractor, server.taskManager)

	// Restore in-memory graph from persisted entities and relationships.
	if err := server.graphEngine.Load(context.Background()); err != nil {
		log.Printf("graph: failed to load persisted graph: %v", err)
	}

	// Initialize agents (coach + simulator participate in forum debates)
	coachAgent := agent.NewCoachAgent(llmClient, store)
	simulatorAgent := agent.NewSimulatorAgent(llmClient)
	server.coachAgent = coachAgent

	agents := []agent.Agent{
		agent.NewEmotionAgent(llmClient),
		agent.NewOpportunityAgent(llmClient),
		agent.NewRiskAgent(llmClient),
		coachAgent,
		simulatorAgent,
	}
	moderator := forum.NewModerator(llmClient)
	server.forumEngine = forum.NewEngine(agents, moderator, store, server.taskManager)

	// Initialize report generator
	reportTools := []report.Tool{
		report.NewGraphSearchTool(store),
		report.NewForumSearchTool(store),
		report.NewMessageSearchTool(store),
	}
	server.reportGenerator = report.NewGenerator(llmClient, store, server.taskManager, reportTools)
	server.renderer = report.NewRenderer()

	// Initialize simulation engine
	profiler := simulation.NewProfileBuilder(llmClient, store)
	server.simEngine = simulation.NewEngine(llmClient, store, server.taskManager, profiler, server.graphEngine.GraphStore())

	// Initialize reminder service
	server.reminderService = reminder.NewService(store)
	server.reminderService.Start(context.Background())

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
		// Stop reminder scheduler first
		if s.reminderService != nil {
			s.reminderService.Stop()
		}
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

// llmSettingKeys are the DB keys used for LLM configuration.
var llmSettingKeys = []string{
	"llm_provider", "llm_base_url", "llm_model", "llm_api_key", "llm_embedding_model",
}

// overrideLLMConfigFromDB reads saved LLM settings from the database and
// overrides the corresponding fields in the given config. Missing keys are
// left unchanged.
func overrideLLMConfigFromDB(store *storage.Store, base llm.Config) llm.Config {
	ctx := context.Background()
	settings, err := store.GetSettings(ctx, llmSettingKeys)
	if err != nil {
		log.Printf("llm-config: failed to load DB settings: %v", err)
		return base
	}
	if len(settings) == 0 {
		return base
	}
	if v, ok := settings["llm_provider"]; ok {
		base.Provider = v
	}
	if v, ok := settings["llm_base_url"]; ok {
		base.BaseURL = v
	}
	if v, ok := settings["llm_model"]; ok {
		base.Model = v
	}
	if v, ok := settings["llm_api_key"]; ok {
		base.APIKey = v
	}
	if v, ok := settings["llm_embedding_model"]; ok {
		base.EmbeddingModel = v
	}
	return base
}

// swapLLMClient rebuilds the LLM client from DB settings (falling back to
// the original env config) and hot-swaps it into the running server.
func (s *Server) swapLLMClient(ctx context.Context) error {
	base := llm.Config{
		Provider:       s.config.LLMProvider,
		BaseURL:        s.config.LLMBaseURL,
		Model:          s.config.LLMModel,
		EmbeddingModel: s.config.EmbeddingModel,
		Timeout:        120 * time.Second,
		APIKey:         s.config.LLMAPIKey,
	}
	cfg := overrideLLMConfigFromDB(s.store, base)

	newClient, err := llm.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("swap LLM client: %w", err)
	}

	s.llmClient.Swap(newClient)
	log.Printf("llm-config: swapped to provider=%s url=%s model=%s", cfg.Provider, cfg.BaseURL, cfg.Model)
	return nil
}
