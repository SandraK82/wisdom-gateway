package models

import "time"

// RelationType defines the kind of relationship between entities.
type RelationType string

const (
	// Trust - the most important relation type
	// Agent trusts/distrusts the From element (Fragment or Agent)
	RelationTrust RelationType = "TRUST"

	// Content relationships
	RelationSupports    RelationType = "SUPPORTS"     // From supports To
	RelationContradicts RelationType = "CONTRADICTS"  // From contradicts To
	RelationExtends     RelationType = "EXTENDS"      // From extends To
	RelationSupersedes  RelationType = "SUPERSEDES"   // From supersedes To
	RelationDerivedFrom RelationType = "DERIVED_FROM" // From is derived from To
	RelationRelatedTo   RelationType = "RELATED_TO"   // Generic relation (also used for tag application)
	RelationExampleOf   RelationType = "EXAMPLE_OF"   // From is an example of To

	// Refinement relations
	RelationSpecializes RelationType = "SPECIALIZES" // From specializes To
	RelationClarifies   RelationType = "CLARIFIES"   // From clarifies To
	RelationGeneralizes RelationType = "GENERALIZES" // From generalizes To

	// Note: Fragment typing (QUESTION, HYPOTHESIS, etc.) now uses TYPE tags
	// instead of self-referencing relations. See wisdom_type_fragment tool.
)

// Relation represents a relationship between entities in the network.
// Relations can express content relationships, trust, or type fragments.
type Relation struct {
	UUID       string       `json:"uuid"`
	From       Address      `json:"from"`       // Required: source entity
	To         Address      `json:"to"`         // Optional: target entity (empty for self-reference)
	By         Address      `json:"by"`         // Agent who asserts this relation
	Type       RelationType `json:"type"`       // Type of relationship
	Content    string       `json:"content"`    // Optional: reasoning/explanation
	Creator    Address      `json:"creator"`    // Agent who created this relation
	Version    uint32       `json:"version"`    // Incremented on updates
	When       time.Time    `json:"when"`       // Creation timestamp
	Signature  string       `json:"signature"`  // Signature over the relation data
	Confidence float32      `json:"confidence"` // Strength of relationship (0.0 to 1.0)
	CreatedAt  time.Time    `json:"created_at"` // Database creation timestamp
}

// Validate checks if the Relation data is well-formed.
func (r *Relation) Validate() error {
	if r.UUID == "" {
		return NewValidationError("relation", "uuid is required")
	}
	if r.From.IsEmpty() {
		return NewValidationError("relation", "from is required")
	}
	if err := r.From.Validate(); err != nil {
		return NewValidationError("relation", "invalid from: %v", err)
	}
	
	// To is optional for certain relation types
	if !r.To.IsEmpty() {
		if err := r.To.Validate(); err != nil {
			return NewValidationError("relation", "invalid to: %v", err)
		}
	}
	
	if r.By.IsEmpty() {
		return NewValidationError("relation", "by is required")
	}
	if err := r.By.Validate(); err != nil {
		return NewValidationError("relation", "invalid by: %v", err)
	}
	if r.By.Domain != DomainAgent {
		return NewValidationError("relation", "by must have AGENT domain")
	}
	
	if r.Type == "" {
		return NewValidationError("relation", "type is required")
	}
	if !isValidRelationType(r.Type) {
		return NewValidationError("relation", "invalid type: %s", r.Type)
	}
	
	if r.Creator.IsEmpty() {
		return NewValidationError("relation", "creator is required")
	}
	if err := r.Creator.Validate(); err != nil {
		return NewValidationError("relation", "invalid creator: %v", err)
	}
	
	if r.When.IsZero() {
		return NewValidationError("relation", "when is required")
	}
	if r.Signature == "" {
		return NewValidationError("relation", "signature is required")
	}
	
	return nil
}

// IsSelfReference returns true if this relation is a self-reference (typing relation).
func (r *Relation) IsSelfReference() bool {
	return r.To.IsEmpty() || r.From.String() == r.To.String()
}

// isValidRelationType checks if a relation type is valid.
func isValidRelationType(rt RelationType) bool {
	switch rt {
	case RelationTrust,
		RelationSupports, RelationContradicts, RelationExtends, RelationSupersedes,
		RelationDerivedFrom, RelationRelatedTo, RelationExampleOf,
		RelationSpecializes, RelationClarifies, RelationGeneralizes:
		return true
	}
	return false
}

// ValidRelationTypes returns all valid relation types.
func ValidRelationTypes() []RelationType {
	return []RelationType{
		RelationTrust,
		RelationSupports, RelationContradicts, RelationExtends, RelationSupersedes,
		RelationDerivedFrom, RelationRelatedTo, RelationExampleOf,
		RelationSpecializes, RelationClarifies, RelationGeneralizes,
	}
}
