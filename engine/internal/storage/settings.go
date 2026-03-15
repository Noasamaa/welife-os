package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// GetSetting retrieves a single setting value by key.
// Returns empty string and sql.ErrNoRows when the key does not exist.
func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM system_settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

// SaveSetting upserts a key-value pair into system_settings.
func (s *Store) SaveSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO system_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	if err != nil {
		return fmt.Errorf("save setting %q: %w", key, err)
	}
	return nil
}

// DeleteSetting removes a setting by key.
func (s *Store) DeleteSetting(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM system_settings WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("delete setting %q: %w", key, err)
	}
	return nil
}

// GetSettings retrieves multiple settings by keys. Missing keys are omitted.
func (s *Store) GetSettings(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		var value string
		err := s.db.QueryRowContext(ctx, "SELECT value FROM system_settings WHERE key = ?", key).Scan(&value)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("get setting %q: %w", key, err)
		}
		result[key] = value
	}
	return result, nil
}
