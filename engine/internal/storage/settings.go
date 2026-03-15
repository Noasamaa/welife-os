package storage

import (
	"context"
	"fmt"
	"strings"
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

// GetSettings retrieves multiple settings by keys in a single query.
// Missing keys are omitted from the result.
func (s *Store) GetSettings(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
	}

	// Build "SELECT key, value FROM system_settings WHERE key IN (?, ?, ...)"
	placeholders := make([]string, len(keys))
	args := make([]interface{}, len(keys))
	for i, k := range keys {
		placeholders[i] = "?"
		args[i] = k
	}
	query := "SELECT key, value FROM system_settings WHERE key IN (" + strings.Join(placeholders, ", ") + ")"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string, len(keys))
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("get settings scan: %w", err)
		}
		result[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get settings rows: %w", err)
	}
	return result, nil
}
