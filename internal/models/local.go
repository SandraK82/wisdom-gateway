package models

import "time"

// Visibility indicates whether a project's data is synced to the hub.
type Visibility string

const (
	// VisibilityPublic means data is synced to the hub.
	VisibilityPublic Visibility = "public"
	// VisibilityPrivate means data stays local.
	VisibilityPrivate Visibility = "private"
)

// Session represents a Gateway-local authentication session.
// Sessions are not federated - they exist only on the local gateway.
type Session struct {
	ID        string    `json:"id"`         // Session UUID
	AgentUUID string    `json:"agent_uuid"` // The authenticated agent
	CreatedAt time.Time `json:"created_at"` // When the session was created
	ExpiresAt time.Time `json:"expires_at"` // When the session expires
	LastUsed  time.Time `json:"last_used"`  // Last activity timestamp
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Validate checks if the Session data is well-formed.
func (s *Session) Validate() error {
	if s.ID == "" {
		return NewValidationError("session", "id is required")
	}
	if s.AgentUUID == "" {
		return NewValidationError("session", "agent_uuid is required")
	}
	if s.CreatedAt.IsZero() {
		return NewValidationError("session", "created_at is required")
	}
	if s.ExpiresAt.IsZero() {
		return NewValidationError("session", "expires_at is required")
	}
	return nil
}

// AuthChallenge represents a challenge for authentication.
// The agent must sign the challenge with their private key.
type AuthChallenge struct {
	ID        string    `json:"id"`         // Challenge UUID
	AgentUUID string    `json:"agent_uuid"` // Agent being challenged
	Challenge string    `json:"challenge"`  // Random challenge string
	CreatedAt time.Time `json:"created_at"` // When challenge was created
	ExpiresAt time.Time `json:"expires_at"` // Challenge expiry time
}

// IsExpired returns true if the challenge has expired.
func (c *AuthChallenge) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// Validate checks if the AuthChallenge data is well-formed.
func (c *AuthChallenge) Validate() error {
	if c.ID == "" {
		return NewValidationError("auth_challenge", "id is required")
	}
	if c.AgentUUID == "" {
		return NewValidationError("auth_challenge", "agent_uuid is required")
	}
	if c.Challenge == "" {
		return NewValidationError("auth_challenge", "challenge is required")
	}
	if c.CreatedAt.IsZero() {
		return NewValidationError("auth_challenge", "created_at is required")
	}
	if c.ExpiresAt.IsZero() {
		return NewValidationError("auth_challenge", "expires_at is required")
	}
	return nil
}

// Project represents a Gateway-local project context.
// Projects help organize fragments and provide context for queries.
type Project struct {
	ID          string     `json:"id"`          // Project UUID
	Name        string     `json:"name"`        // Human-readable name
	Description string     `json:"description"` // Project description
	AgentUUID   string     `json:"agent_uuid"`  // Owner agent
	Tags        []string   `json:"tags"`        // Default tags (UUIDs)
	Visibility  Visibility `json:"visibility"`  // public or private
	CreatedAt   time.Time  `json:"created_at"`  // Creation timestamp
	UpdatedAt   time.Time  `json:"updated_at"`  // Last update timestamp
}

// Validate checks if the Project data is well-formed.
func (p *Project) Validate() error {
	if p.ID == "" {
		return NewValidationError("project", "id is required")
	}
	if p.Name == "" {
		return NewValidationError("project", "name is required")
	}
	if p.AgentUUID == "" {
		return NewValidationError("project", "agent_uuid is required")
	}
	if p.CreatedAt.IsZero() {
		return NewValidationError("project", "created_at is required")
	}
	return nil
}
