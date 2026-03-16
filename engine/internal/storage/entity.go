package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// ClearGraph removes all entities and relationships from the database.
func (s *Store) ClearGraph(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM relationships`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM entities`); err != nil {
		return err
	}
	return tx.Commit()
}

// ClearGraphForConversation removes only entities and relationships belonging
// to a specific conversation, leaving other conversations' data intact.
func (s *Store) ClearGraphForConversation(ctx context.Context, conversationID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Delete relationships where both source and target belong to this conversation
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM relationships WHERE id IN (
			SELECT r.id FROM relationships r
			INNER JOIN entities src ON src.id = r.source_entity_id
			INNER JOIN entities dst ON dst.id = r.target_entity_id
			WHERE src.source_conversation = ? OR dst.source_conversation = ?
		)`, conversationID, conversationID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM entities WHERE source_conversation = ?`, conversationID); err != nil {
		return err
	}
	return tx.Commit()
}

// SaveEntity inserts or replaces an entity.
func (s *Store) SaveEntity(ctx context.Context, e Entity) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO entities (id, type, name, properties, source_conversation)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name = excluded.name, properties = excluded.properties`,
		e.ID, e.Type, e.Name, e.Properties, e.SourceConversation)
	return err
}

// SaveEntities inserts multiple entities in a transaction.
func (s *Store) SaveEntities(ctx context.Context, entities []Entity) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO entities (id, type, name, properties, source_conversation)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name = excluded.name, properties = excluded.properties`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entities {
		if _, err := stmt.ExecContext(ctx, e.ID, e.Type, e.Name, e.Properties, e.SourceConversation); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetEntity returns a single entity by ID.
// Returns an error containing "not found" if no entity matches.
func (s *Store) GetEntity(ctx context.Context, id string) (Entity, error) {
	var e Entity
	err := s.db.QueryRowContext(ctx, `
		SELECT id, type, name, COALESCE(properties,''), COALESCE(source_conversation,'')
		FROM entities WHERE id = ?`, id).
		Scan(&e.ID, &e.Type, &e.Name, &e.Properties, &e.SourceConversation)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Entity{}, fmt.Errorf("entity %s: %w", id, ErrNotFound)
		}
		return Entity{}, err
	}
	return e, nil
}

// SearchEntitiesByName returns entities whose name matches the query
// (case-insensitive substring match), limited to at most maxResults rows.
func (s *Store) SearchEntitiesByName(ctx context.Context, query string, maxResults int) ([]Entity, error) {
	if maxResults <= 0 {
		maxResults = 50
	}
	// Cap query length to prevent DoS via huge LIKE patterns.
	const maxQueryLen = 200
	if len(query) > maxQueryLen {
		query = query[:maxQueryLen]
	}
	// Escape LIKE metacharacters so user input is treated as literal substring.
	escaped := strings.ReplaceAll(query, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "%", "\\%")
	escaped = strings.ReplaceAll(escaped, "_", "\\_")
	pattern := "%" + escaped + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, name, COALESCE(properties,''), COALESCE(source_conversation,'')
		FROM entities WHERE name LIKE ? ESCAPE '\' COLLATE NOCASE
		ORDER BY name LIMIT ?`, pattern, maxResults)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.ID, &e.Type, &e.Name, &e.Properties, &e.SourceConversation); err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

// FindEntitiesByType returns all entities of a given type.
func (s *Store) FindEntitiesByType(ctx context.Context, entityType string) ([]Entity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, name, COALESCE(properties,''), COALESCE(source_conversation,'')
		FROM entities WHERE type = ? ORDER BY name`, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.ID, &e.Type, &e.Name, &e.Properties, &e.SourceConversation); err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

// FindEntitiesByTypeInConversation returns all entities of a given type
// extracted from a specific conversation.
func (s *Store) FindEntitiesByTypeInConversation(ctx context.Context, entityType, conversationID string) ([]Entity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, name, COALESCE(properties,''), COALESCE(source_conversation,'')
		FROM entities WHERE type = ? AND source_conversation = ? ORDER BY name`, entityType, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.ID, &e.Type, &e.Name, &e.Properties, &e.SourceConversation); err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

// ListEntities returns all entities.
func (s *Store) ListEntities(ctx context.Context) ([]Entity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, name, COALESCE(properties,''), COALESCE(source_conversation,'')
		FROM entities ORDER BY type, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.ID, &e.Type, &e.Name, &e.Properties, &e.SourceConversation); err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

// ListEntitiesByConversation returns all entities extracted from a single conversation.
func (s *Store) ListEntitiesByConversation(ctx context.Context, conversationID string) ([]Entity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, name, COALESCE(properties,''), COALESCE(source_conversation,'')
		FROM entities WHERE source_conversation = ? ORDER BY type, name`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.ID, &e.Type, &e.Name, &e.Properties, &e.SourceConversation); err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

// SaveRelationships inserts multiple relationships in a transaction.
func (s *Store) SaveRelationships(ctx context.Context, rels []Relationship) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO relationships (id, source_entity_id, target_entity_id, type, properties, weight, source_message_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET weight = excluded.weight`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range rels {
		if _, err := stmt.ExecContext(ctx, r.ID, r.SourceEntityID, r.TargetEntityID,
			r.Type, r.Properties, r.Weight, r.SourceMessageID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetRelationships returns all relationships for a given entity (as source or target).
func (s *Store) GetRelationships(ctx context.Context, entityID string) ([]Relationship, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source_entity_id, target_entity_id, type, COALESCE(properties,''),
		       weight, COALESCE(source_message_id,'')
		FROM relationships
		WHERE source_entity_id = ? OR target_entity_id = ?
		ORDER BY weight DESC`, entityID, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []Relationship
	for rows.Next() {
		var r Relationship
		if err := rows.Scan(&r.ID, &r.SourceEntityID, &r.TargetEntityID,
			&r.Type, &r.Properties, &r.Weight, &r.SourceMessageID); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

// ListRelationships returns all relationships.
func (s *Store) ListRelationships(ctx context.Context) ([]Relationship, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source_entity_id, target_entity_id, type, COALESCE(properties,''),
		       weight, COALESCE(source_message_id,'')
		FROM relationships ORDER BY weight DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []Relationship
	for rows.Next() {
		var r Relationship
		if err := rows.Scan(&r.ID, &r.SourceEntityID, &r.TargetEntityID,
			&r.Type, &r.Properties, &r.Weight, &r.SourceMessageID); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

// ListRelationshipsByConversation returns relationships whose source and target
// entities both belong to the given conversation.
func (s *Store) ListRelationshipsByConversation(ctx context.Context, conversationID string) ([]Relationship, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.source_entity_id, r.target_entity_id, r.type,
		       COALESCE(r.properties,''), r.weight, COALESCE(r.source_message_id,'')
		FROM relationships r
		INNER JOIN entities src ON src.id = r.source_entity_id
		INNER JOIN entities dst ON dst.id = r.target_entity_id
		WHERE src.source_conversation = ? AND dst.source_conversation = ?
		ORDER BY r.weight DESC`, conversationID, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []Relationship
	for rows.Next() {
		var r Relationship
		if err := rows.Scan(&r.ID, &r.SourceEntityID, &r.TargetEntityID,
			&r.Type, &r.Properties, &r.Weight, &r.SourceMessageID); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}
