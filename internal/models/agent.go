package models

// Bias represents a known bias or tendency of an agent.
type Bias struct {
	Domain      string  `json:"domain"`      // Domain this bias applies to
	Description string  `json:"description"` // Description of the bias
	Severity    float32 `json:"severity"`    // Severity (0.0 to 1.0)
}

// AgentProfile contains an agent's expertise and characteristics.
type AgentProfile struct {
	Specializations    map[string]float32 `json:"specializations"`     // Expertise per domain (0.0 to 1.0)
	KnownBiases        []Bias             `json:"known_biases"`        // Known biases or tendencies
	AvgConfidence      float32            `json:"avg_confidence"`      // Average confidence in created fragments
	FragmentCount      uint64             `json:"fragment_count"`      // Total fragments created
	HistoricalAccuracy float32            `json:"historical_accuracy"` // Historical accuracy score (0.0 to 1.0)
}

// NewAgentProfile creates a new empty profile.
func NewAgentProfile() AgentProfile {
	return AgentProfile{
		Specializations: make(map[string]float32),
		KnownBiases:     []Bias{},
	}
}

// AddSpecialization adds or updates a specialization score.
func (p *AgentProfile) AddSpecialization(domain string, score float32) {
	if score < 0 {
		score = 0
	} else if score > 1 {
		score = 1
	}
	p.Specializations[domain] = score
}

// GetSpecialization returns the specialization score for a domain.
func (p *AgentProfile) GetSpecialization(domain string) float32 {
	if score, ok := p.Specializations[domain]; ok {
		return score
	}
	return 0
}

// UpdateStats updates the profile statistics after creating a fragment.
func (p *AgentProfile) UpdateStats(confidence float32) {
	total := float32(p.FragmentCount)*p.AvgConfidence + confidence
	p.FragmentCount++
	p.AvgConfidence = total / float32(p.FragmentCount)
}

// Agent represents a participant in the Shared-Wisdom network.
// An agent can create fragments, relations, and express trust in other agents.
type Agent struct {
	UUID        string       `json:"uuid"`
	PublicKey   string       `json:"public_key"`   // Base64-encoded public key
	Version     uint32       `json:"version"`      // Incremented on updates
	Description string       `json:"description"`  // Human-readable description
	Trust       TrustStore   `json:"trust"`        // Embedded trust relationships for path finding
	PrimaryHub  Address      `json:"primary_hub"`  // Where this agent's data primarily lives
	Signature   string       `json:"signature"`    // Signature over the agent data
	Profile     AgentProfile `json:"profile"`      // Agent's expertise profile
}

// TrustStore contains an agent's direct trust relationships.
// This is embedded in the Agent for efficient trust path calculations.
type TrustStore struct {
	NumTrusts uint64  `json:"num_trusts"` // Total number of trust relationships
	Trusts    []Trust `json:"trusts"`     // Direct trust relationships
}

// Trust represents a direct trust relationship from one agent to another.
type Trust struct {
	Agent Address `json:"agent"` // The agent being trusted/distrusted
	Trust float32 `json:"trust"` // Trust level: -1.0 (distrust) to 1.0 (full trust)
}

// Validate checks if the Agent data is well-formed.
func (a *Agent) Validate() error {
	if a.UUID == "" {
		return NewValidationError("agent", "uuid is required")
	}
	if a.PublicKey == "" {
		return NewValidationError("agent", "public_key is required")
	}
	if a.Signature == "" {
		return NewValidationError("agent", "signature is required")
	}
	
	// Validate embedded trust addresses
	for i, t := range a.Trust.Trusts {
		if err := t.Agent.Validate(); err != nil {
			return NewValidationError("agent", "invalid trust[%d].agent: %v", i, err)
		}
		if t.Trust < -1.0 || t.Trust > 1.0 {
			return NewValidationError("agent", "trust[%d].trust must be between -1.0 and 1.0", i)
		}
	}
	
	// Validate primary hub if set
	if !a.PrimaryHub.IsEmpty() {
		if err := a.PrimaryHub.Validate(); err != nil {
			return NewValidationError("agent", "invalid primary_hub: %v", err)
		}
	}
	
	return nil
}

// GetTrustFor returns the trust value for a specific agent, or 0 if not found.
func (a *Agent) GetTrustFor(agentAddr Address) float32 {
	for _, t := range a.Trust.Trusts {
		if t.Agent.String() == agentAddr.String() {
			return t.Trust
		}
	}
	return 0
}
