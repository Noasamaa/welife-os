package storage

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

type Config struct {
	Path string
	Key  string
}

type Store struct {
	db   *sql.DB
	path string
}

func Open(ctx context.Context, cfg Config) (*Store, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("database path is required")
	}
	if cfg.Key == "" {
		return nil, fmt.Errorf("database key is required")
	}

	absPath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf(
		"%s?_pragma_key=%s&_pragma_cipher_page_size=4096&_pragma_foreign_keys=ON&_busy_timeout=5000",
		absPath,
		url.QueryEscape(cfg.Key),
	)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	store := &Store{db: db, path: absPath}
	if err := store.initialize(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) initialize(ctx context.Context) error {
	for _, stmt := range schemaStatements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return s.Probe(ctx)
}

func (s *Store) Probe(ctx context.Context) error {
	var result int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master;").Scan(&result); err != nil {
		return err
	}
	return nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Driver() string {
	return "sqlcipher"
}

func (s *Store) Close() error {
	return s.db.Close()
}
