// Package models contains the core data structures for the Shared-Wisdom gateway.
package models

import (
	"errors"
	"fmt"
	"strings"
)

// AddressDomain represents the type of entity an Address points to.
type AddressDomain string

const (
	DomainAgent          AddressDomain = "AGENT"
	DomainTag            AddressDomain = "TAG"
	DomainFragment       AddressDomain = "FRAGMENT"
	DomainRelation       AddressDomain = "RELATION"
	DomainTransformation AddressDomain = "TRANSFORMATION"
	DomainHub            AddressDomain = "HUB"
)

// Address is a federated identifier for any entity in the network.
// Format: "server:port/DOMAIN/entity-uuid"
type Address struct {
	ServerPort string        `json:"server_port"` // FQDN:port (e.g., "hub.example.com:8443")
	Domain     AddressDomain `json:"domain"`      // Entity type
	Entity     string        `json:"entity"`      // UUID of the entity
}

// String returns the canonical string representation of an Address.
func (a Address) String() string {
	if a.ServerPort == "" && a.Domain == "" && a.Entity == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s", a.ServerPort, a.Domain, a.Entity)
}

// IsEmpty returns true if the Address is unset.
func (a Address) IsEmpty() bool {
	return a.ServerPort == "" && a.Domain == "" && a.Entity == ""
}

// IsLocal returns true if the Address points to a local entity (no server specified).
func (a Address) IsLocal() bool {
	return a.ServerPort == ""
}

// ParseAddress parses a string into an Address.
// Format: "server:port/DOMAIN/entity-uuid" or "/DOMAIN/entity-uuid" for local addresses.
func ParseAddress(s string) (Address, error) {
	if s == "" {
		return Address{}, nil
	}

	parts := strings.Split(s, "/")
	
	// Local address: /DOMAIN/entity
	if strings.HasPrefix(s, "/") && len(parts) == 3 {
		domain := AddressDomain(parts[1])
		if !isValidDomain(domain) {
			return Address{}, fmt.Errorf("invalid domain: %s", parts[1])
		}
		return Address{
			ServerPort: "",
			Domain:     domain,
			Entity:     parts[2],
		}, nil
	}
	
	// Remote address: server:port/DOMAIN/entity
	if len(parts) == 3 {
		domain := AddressDomain(parts[1])
		if !isValidDomain(domain) {
			return Address{}, fmt.Errorf("invalid domain: %s", parts[1])
		}
		return Address{
			ServerPort: parts[0],
			Domain:     domain,
			Entity:     parts[2],
		}, nil
	}

	return Address{}, errors.New("invalid address format: expected 'server:port/DOMAIN/entity' or '/DOMAIN/entity'")
}

// Validate checks if the Address is well-formed.
func (a Address) Validate() error {
	if a.IsEmpty() {
		return nil // Empty address is valid (optional field)
	}
	
	if a.Domain == "" {
		return errors.New("address domain is required")
	}
	
	if !isValidDomain(a.Domain) {
		return fmt.Errorf("invalid address domain: %s", a.Domain)
	}
	
	if a.Entity == "" {
		return errors.New("address entity is required")
	}
	
	return nil
}

// isValidDomain checks if a domain string is one of the known domains.
func isValidDomain(d AddressDomain) bool {
	switch d {
	case DomainAgent, DomainTag, DomainFragment, DomainRelation, DomainTransformation, DomainHub:
		return true
	}
	return false
}

// ValidDomains returns a list of all valid domain values.
func ValidDomains() []AddressDomain {
	return []AddressDomain{
		DomainAgent,
		DomainTag,
		DomainFragment,
		DomainRelation,
		DomainTransformation,
		DomainHub,
	}
}
