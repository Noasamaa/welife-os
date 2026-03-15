package forum

// SessionStatus represents the lifecycle state of a debate session.
type SessionStatus string

const (
	StatusRunning   SessionStatus = "running"
	StatusCompleted SessionStatus = "completed"
	StatusFailed    SessionStatus = "failed"
)

// DebateConfig controls the debate execution parameters.
type DebateConfig struct {
	DebateRounds int // number of cross-debate rounds (default: 2)
}

// DefaultConfig returns the default debate configuration.
func DefaultConfig() DebateConfig {
	return DebateConfig{
		DebateRounds: 2,
	}
}
