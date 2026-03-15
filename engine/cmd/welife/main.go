package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
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
	dbPath := lookupString("WELIFE_DB_PATH", defaultDBPath)
	dbKey, err := resolveDatabaseKey(dbPath)
	if err != nil {
		return server.Config{}, err
	}

	port, err := lookupInt("WELIFE_PORT", 18080)
	if err != nil {
		return server.Config{}, err
	}

	return server.Config{
		Host:         lookupString("WELIFE_HOST", "127.0.0.1"),
		Port:         port,
		DatabasePath: dbPath,
		DatabaseKey:  dbKey,
		LLMProvider:  lookupString("WELIFE_LLM_PROVIDER", "ollama"),
		LLMBaseURL:   lookupString("WELIFE_OLLAMA_BASE_URL", "http://127.0.0.1:11434"),
		LLMModel:     lookupString("WELIFE_OLLAMA_MODEL", "qwen3.5:9b"),
		LLMAPIKey:    lookupString("WELIFE_LLM_API_KEY", ""),
	}, nil
}

func resolveDatabaseKey(databasePath string) (string, error) {
	if value, ok := os.LookupEnv("WELIFE_DB_KEY"); ok && value != "" {
		return value, nil
	}

	keyPath := filepath.Join(filepath.Dir(databasePath), "welife.key")
	if raw, err := os.ReadFile(keyPath); err == nil {
		if key := strings.TrimSpace(string(raw)); key != "" {
			return key, nil
		}
		return "", fmt.Errorf("database key file %s is empty", keyPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return "", err
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	key := hex.EncodeToString(secret)
	if err := os.WriteFile(keyPath, []byte(key+"\n"), 0o600); err != nil {
		return "", err
	}
	return key, nil
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
