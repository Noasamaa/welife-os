package simulation

// ForkPoint describes the divergence point for a simulation.
type ForkPoint struct {
	Description   string            `json:"description"`
	AffectedNodes []string          `json:"affected_nodes"`
	Changes       map[string]string `json:"changes"`
}

// SimulationConfig holds the parameters for running a simulation.
type SimulationConfig struct {
	Steps     int       `json:"steps"`
	ForkPoint ForkPoint `json:"fork_point"`
}
