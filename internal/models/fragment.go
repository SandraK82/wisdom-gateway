package models

import "time"

// EvidenceType indicates how the fragment's content was derived.
type EvidenceType string

const (
	// EvidenceEmpirical - observed or tested empirically
	EvidenceEmpirical EvidenceType = "empirical"
	// EvidenceLogical - logically derived from other facts
	EvidenceLogical EvidenceType = "logical"
	// EvidenceConsensus - agreed upon by multiple sources
	EvidenceConsensus EvidenceType = "consensus"
	// EvidenceSpeculation - hypothetical/speculative
	EvidenceSpeculation EvidenceType = "speculation"
	// EvidenceUnknown - not specified (default)
	EvidenceUnknown EvidenceType = "unknown"
)

// IsValid checks if the evidence type is valid.
func (e EvidenceType) IsValid() bool {
	switch e {
	case EvidenceEmpirical, EvidenceLogical, EvidenceConsensus, EvidenceSpeculation, EvidenceUnknown:
		return true
	}
	return false
}

// Fragment represents a piece of knowledge in the Shared-Wisdom network.
// Fragments are minimal - typing and state are expressed through Relations.
type Fragment struct {
	UUID      string    `json:"uuid"`
	Tags      []Address `json:"tags"`      // References to Tag entities
	Transform Address   `json:"transform"` // Reference to Transform entity
	Content   string    `json:"content"`   // The actual content
	Creator      Address      `json:"creator"`       // Agent who created this fragment
	When         time.Time    `json:"when"`          // Creation timestamp
	Signature    string       `json:"signature"`     // Signature over the fragment data
	Confidence   float32      `json:"confidence"`    // Creator's confidence (0.0 to 1.0)
	EvidenceType EvidenceType `json:"evidence_type"` // How the content was derived
}

// Validate checks if the Fragment data is well-formed.
func (f *Fragment) Validate() error {
	if f.UUID == "" {
		return NewValidationError("fragment", "uuid is required")
	}
	if f.Content == "" {
		return NewValidationError("fragment", "content is required")
	}
	if f.Creator.IsEmpty() {
		return NewValidationError("fragment", "creator is required")
	}
	if err := f.Creator.Validate(); err != nil {
		return NewValidationError("fragment", "invalid creator: %v", err)
	}
	if f.When.IsZero() {
		return NewValidationError("fragment", "when is required")
	}
	if f.Signature == "" {
		return NewValidationError("fragment", "signature is required")
	}
	
	// Validate tag addresses
	for i, tag := range f.Tags {
		if err := tag.Validate(); err != nil {
			return NewValidationError("fragment", "invalid tags[%d]: %v", i, err)
		}
		if tag.Domain != DomainTag {
			return NewValidationError("fragment", "tags[%d] must have TAG domain", i)
		}
	}
	
	// Validate transform address if set
	if !f.Transform.IsEmpty() {
		if err := f.Transform.Validate(); err != nil {
			return NewValidationError("fragment", "invalid transform: %v", err)
		}
		if f.Transform.Domain != DomainTransformation {
			return NewValidationError("fragment", "transform must have TRANSFORMATION domain")
		}
	}
	
	return nil
}

// HasTag returns true if the fragment has a tag with the given UUID.
func (f *Fragment) HasTag(tagUUID string) bool {
	for _, tag := range f.Tags {
		if tag.Entity == tagUUID {
			return true
		}
	}
	return false
}
