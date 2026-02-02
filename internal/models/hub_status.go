package models

// ResourceLevel represents the resource usage level of a hub
type ResourceLevel string

const (
	ResourceLevelNormal   ResourceLevel = "normal"
	ResourceLevelWarning  ResourceLevel = "warning"
	ResourceLevelCritical ResourceLevel = "critical"
)

// HubStatus represents the resource status of the upstream hub
type HubStatus struct {
	Level    ResourceLevel `json:"level"`
	Hint     string        `json:"hint,omitempty"`
	Warnings []string      `json:"warnings,omitempty"`
}

// IsNormal returns true if the hub is operating normally
func (h *HubStatus) IsNormal() bool {
	return h == nil || h.Level == ResourceLevelNormal || h.Level == ""
}

// IsCritical returns true if the hub is at critical capacity
func (h *HubStatus) IsCritical() bool {
	return h != nil && h.Level == ResourceLevelCritical
}

// IsWarning returns true if the hub is at warning level
func (h *HubStatus) IsWarning() bool {
	return h != nil && h.Level == ResourceLevelWarning
}
