package storage

import "context"

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
