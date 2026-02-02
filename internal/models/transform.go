package models

// Transform defines how content is transformed between formats.
// This is used to interpret fragment content consistently.
type Transform struct {
	UUID           string    `json:"uuid"`
	Name           string    `json:"name"`           // Human-readable name
	Description    string    `json:"description"`    // What this transform does
	Tags           []Address `json:"tags"`           // Related tags
	TransformTo    string    `json:"transform_to"`   // Target format (e.g., "text/markdown")
	TransformFrom  string    `json:"transform_from"` // Source format (e.g., "text/plain")
	AdditionalData string    `json:"additional_data"` // JSON with extra configuration
	Agent          Address   `json:"agent"`          // Agent who created this transform
	Version        uint32    `json:"version"`        // Incremented on updates
	Signature      string    `json:"signature"`      // Signature over the transform data
}

// Validate checks if the Transform data is well-formed.
func (t *Transform) Validate() error {
	if t.UUID == "" {
		return NewValidationError("transform", "uuid is required")
	}
	if t.Name == "" {
		return NewValidationError("transform", "name is required")
	}
	if t.TransformTo == "" {
		return NewValidationError("transform", "transform_to is required")
	}
	if t.TransformFrom == "" {
		return NewValidationError("transform", "transform_from is required")
	}
	if t.Agent.IsEmpty() {
		return NewValidationError("transform", "agent is required")
	}
	if err := t.Agent.Validate(); err != nil {
		return NewValidationError("transform", "invalid agent: %v", err)
	}
	if t.Agent.Domain != DomainAgent {
		return NewValidationError("transform", "agent must have AGENT domain")
	}
	if t.Signature == "" {
		return NewValidationError("transform", "signature is required")
	}
	
	// Validate tag addresses
	for i, tag := range t.Tags {
		if err := tag.Validate(); err != nil {
			return NewValidationError("transform", "invalid tags[%d]: %v", i, err)
		}
		if tag.Domain != DomainTag {
			return NewValidationError("transform", "tags[%d] must have TAG domain", i)
		}
	}
	
	return nil
}
