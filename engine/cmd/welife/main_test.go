package main

import (
	"path/filepath"
	"testing"
)

func TestResolveDatabaseKeyPersistsGeneratedKey(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "welife.db")

	key1, err := resolveDatabaseKey(dbPath)
	if err != nil {
		t.Fatalf("resolve first key: %v", err)
	}
	if key1 == "" {
		t.Fatal("expected generated key")
	}

	key2, err := resolveDatabaseKey(dbPath)
	if err != nil {
		t.Fatalf("resolve second key: %v", err)
	}
	if key1 != key2 {
		t.Fatalf("expected persisted key, got %q then %q", key1, key2)
	}
}

func TestResolveDatabaseKeyUsesEnvOverride(t *testing.T) {
	t.Setenv("WELIFE_DB_KEY", "override-key")

	key, err := resolveDatabaseKey(filepath.Join(t.TempDir(), "welife.db"))
	if err != nil {
		t.Fatalf("resolve env key: %v", err)
	}
	if key != "override-key" {
		t.Fatalf("expected env override, got %q", key)
	}
}
