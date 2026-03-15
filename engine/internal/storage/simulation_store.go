package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// SavePersonProfile inserts or replaces a person profile.
func (s *Store) SavePersonProfile(ctx context.Context, p PersonProfile) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO person_profiles (id, entity_id, name, personality, relationship_to_self,
		    behavioral_patterns, source_conversation_ids, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
		    personality = excluded.personality,
		    relationship_to_self = excluded.relationship_to_self,
		    behavioral_patterns = excluded.behavioral_patterns,
		    source_conversation_ids = excluded.source_conversation_ids,
		    updated_at = CURRENT_TIMESTAMP`,
		p.ID, p.EntityID, p.Name, p.Personality, p.RelationshipToSelf,
		p.BehavioralPatterns, p.SourceConversationIDs)
	return err
}

// ListPersonProfiles returns all person profiles.
func (s *Store) ListPersonProfiles(ctx context.Context) ([]PersonProfile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, entity_id, name, personality, relationship_to_self,
		       COALESCE(behavioral_patterns,''), COALESCE(source_conversation_ids,''),
		       created_at, updated_at
		FROM person_profiles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []PersonProfile
	for rows.Next() {
		var p PersonProfile
		if err := rows.Scan(&p.ID, &p.EntityID, &p.Name, &p.Personality,
			&p.RelationshipToSelf, &p.BehavioralPatterns,
			&p.SourceConversationIDs, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

// ListPersonProfilesByConversation returns all person profiles derived from a
// specific conversation.
func (s *Store) ListPersonProfilesByConversation(ctx context.Context, conversationID string) ([]PersonProfile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, entity_id, name, personality, relationship_to_self,
		       COALESCE(behavioral_patterns,''), COALESCE(source_conversation_ids,''),
		       created_at, updated_at
		FROM person_profiles
		WHERE source_conversation_ids = ?
		ORDER BY name`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []PersonProfile
	for rows.Next() {
		var p PersonProfile
		if err := rows.Scan(&p.ID, &p.EntityID, &p.Name, &p.Personality,
			&p.RelationshipToSelf, &p.BehavioralPatterns,
			&p.SourceConversationIDs, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

// GetPersonProfile returns a person profile by entity ID.
func (s *Store) GetPersonProfile(ctx context.Context, entityID string) (PersonProfile, error) {
	var p PersonProfile
	err := s.db.QueryRowContext(ctx, `
		SELECT id, entity_id, name, personality, relationship_to_self,
		       COALESCE(behavioral_patterns,''), COALESCE(source_conversation_ids,''),
		       created_at, updated_at
		FROM person_profiles WHERE entity_id = ?`, entityID).
		Scan(&p.ID, &p.EntityID, &p.Name, &p.Personality,
			&p.RelationshipToSelf, &p.BehavioralPatterns,
			&p.SourceConversationIDs, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return p, fmt.Errorf("person profile for entity %q not found", entityID)
	}
	return p, err
}

// CreateSimulationSession inserts a new simulation session.
func (s *Store) CreateSimulationSession(ctx context.Context, sess SimulationSession) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO simulation_sessions (id, conversation_id, task_id, fork_description, status,
		    step_count, original_graph_snapshot, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		sess.ID, sess.ConversationID, sess.TaskID, sess.ForkDescription, sess.Status,
		sess.StepCount, sess.OriginalGraphSnapshot)
	return err
}

// UpdateSimulationSession updates a simulation session's status, narrative, final snapshot, and step count.
func (s *Store) UpdateSimulationSession(ctx context.Context, id, status, narrative, finalSnapshot string, stepCount int) error {
	if status == "completed" || status == "failed" {
		_, err := s.db.ExecContext(ctx, `
			UPDATE simulation_sessions
			SET status = ?, narrative = ?, final_graph_snapshot = ?,
			    step_count = ?, completed_at = CURRENT_TIMESTAMP
			WHERE id = ?`, status, narrative, finalSnapshot, stepCount, id)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE simulation_sessions
		SET status = ?, narrative = ?, final_graph_snapshot = ?, step_count = ?
		WHERE id = ?`, status, narrative, finalSnapshot, stepCount, id)
	return err
}

// GetSimulationSession returns a simulation session by ID.
func (s *Store) GetSimulationSession(ctx context.Context, id string) (SimulationSession, error) {
	var sess SimulationSession
	err := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(conversation_id,''), task_id, fork_description, status, step_count,
		       COALESCE(original_graph_snapshot,''), COALESCE(final_graph_snapshot,''),
		       COALESCE(narrative,''), created_at, COALESCE(completed_at,'')
		FROM simulation_sessions WHERE id = ?`, id).
		Scan(&sess.ID, &sess.ConversationID, &sess.TaskID, &sess.ForkDescription, &sess.Status,
			&sess.StepCount, &sess.OriginalGraphSnapshot, &sess.FinalGraphSnapshot,
			&sess.Narrative, &sess.CreatedAt, &sess.CompletedAt)
	if err == sql.ErrNoRows {
		return sess, fmt.Errorf("simulation session %q not found", id)
	}
	return sess, err
}

// ListSimulationSessions returns all simulation sessions ordered by creation time.
func (s *Store) ListSimulationSessions(ctx context.Context) ([]SimulationSession, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(conversation_id,''), task_id, fork_description, status, step_count,
		       COALESCE(original_graph_snapshot,''), COALESCE(final_graph_snapshot,''),
		       COALESCE(narrative,''), created_at, COALESCE(completed_at,'')
		FROM simulation_sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SimulationSession
	for rows.Next() {
		var sess SimulationSession
		if err := rows.Scan(&sess.ID, &sess.ConversationID, &sess.TaskID, &sess.ForkDescription, &sess.Status,
			&sess.StepCount, &sess.OriginalGraphSnapshot, &sess.FinalGraphSnapshot,
			&sess.Narrative, &sess.CreatedAt, &sess.CompletedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// ListSimulationSessionsByConversation returns all simulation sessions for a
// given conversation ordered by creation time.
func (s *Store) ListSimulationSessionsByConversation(ctx context.Context, conversationID string) ([]SimulationSession, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(conversation_id,''), task_id, fork_description, status, step_count,
		       COALESCE(original_graph_snapshot,''), COALESCE(final_graph_snapshot,''),
		       COALESCE(narrative,''), created_at, COALESCE(completed_at,'')
		FROM simulation_sessions
		WHERE conversation_id = ?
		ORDER BY created_at DESC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SimulationSession
	for rows.Next() {
		var sess SimulationSession
		if err := rows.Scan(&sess.ID, &sess.ConversationID, &sess.TaskID, &sess.ForkDescription, &sess.Status,
			&sess.StepCount, &sess.OriginalGraphSnapshot, &sess.FinalGraphSnapshot,
			&sess.Narrative, &sess.CreatedAt, &sess.CompletedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// SaveSimulationStep inserts a simulation step.
func (s *Store) SaveSimulationStep(ctx context.Context, step SimulationStep) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO simulation_steps (id, session_id, step_number, description,
		    entity_changes, reactions, created_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		step.ID, step.SessionID, step.StepNumber, step.Description,
		step.EntityChanges, step.Reactions)
	return err
}

// GetSimulationSteps returns all steps for a simulation session ordered by step number.
func (s *Store) GetSimulationSteps(ctx context.Context, sessionID string) ([]SimulationStep, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, step_number, description,
		       entity_changes, reactions, created_at
		FROM simulation_steps WHERE session_id = ?
		ORDER BY step_number ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []SimulationStep
	for rows.Next() {
		var step SimulationStep
		if err := rows.Scan(&step.ID, &step.SessionID, &step.StepNumber,
			&step.Description, &step.EntityChanges, &step.Reactions, &step.CreatedAt); err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, rows.Err()
}
