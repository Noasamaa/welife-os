package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/welife-os/welife-os/engine/internal/server"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	app, err := server.New(cfg)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("welife engine listening on http://%s", cfg.Addr())

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	if err := app.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve: %v", err)
	}
}

func loadConfig() (server.Config, error) {
	workdir, err := os.Getwd()
	if err != nil {
		return server.Config{}, err
	}

	defaultDataDir := filepath.Join(workdir, ".data")
	defaultDBPath := filepath.Join(defaultDataDir, "welife.db")

	port, err := lookupInt("WELIFE_PORT", 18080)
	if err != nil {
		return server.Config{}, err
	}

	return server.Config{
		Host:          lookupString("WELIFE_HOST", "127.0.0.1"),
		Port:          port,
		DatabasePath:  lookupString("WELIFE_DB_PATH", defaultDBPath),
		DatabaseKey:   lookupString("WELIFE_DB_KEY", "welife-phase0-dev-key"),
		OllamaBaseURL: lookupString("WELIFE_OLLAMA_BASE_URL", "http://127.0.0.1:11434"),
		OllamaModel:   lookupString("WELIFE_OLLAMA_MODEL", "qwen3.5:9b"),
	}, nil
}

func lookupString(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func lookupInt(key string, fallback int) (int, error) {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	}
	return fallback, nil
}
