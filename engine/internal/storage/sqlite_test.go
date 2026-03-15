package storage_test

import (
	"context"
	"testing"

	sqlite3 "github.com/mutecomm/go-sqlcipher/v4"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

func TestOpenCreatesEncryptedDatabase(t *testing.T) {
	store, err := storage.Open(context.Background(), storage.Config{
		Path: t.TempDir() + "/welife.db",
		Key:  "welife-phase0-test-key",
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	if err := store.Probe(context.Background()); err != nil {
		t.Fatalf("probe: %v", err)
	}

	encrypted, err := sqlite3.IsEncrypted(store.Path())
	if err != nil {
		t.Fatalf("check encrypted: %v", err)
	}
	if !encrypted {
		t.Fatal("database should be encrypted")
	}
}
