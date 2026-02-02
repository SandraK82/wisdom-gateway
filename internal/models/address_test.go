package models

import (
	"testing"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Address
		wantErr bool
	}{
		{
			name:  "empty string",
			input: "",
			want:  Address{},
		},
		{
			name:  "remote address",
			input: "hub.example.com:8443/AGENT/550e8400-e29b-41d4-a716-446655440000",
			want: Address{
				ServerPort: "hub.example.com:8443",
				Domain:     DomainAgent,
				Entity:     "550e8400-e29b-41d4-a716-446655440000",
			},
		},
		{
			name:  "local address",
			input: "/FRAGMENT/550e8400-e29b-41d4-a716-446655440000",
			want: Address{
				ServerPort: "",
				Domain:     DomainFragment,
				Entity:     "550e8400-e29b-41d4-a716-446655440000",
			},
		},
		{
			name:    "invalid domain",
			input:   "hub:8080/INVALID/uuid",
			wantErr: true,
		},
		{
			name:    "malformed",
			input:   "just-a-string",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAddress(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.ServerPort != tt.want.ServerPort {
					t.Errorf("ParseAddress() ServerPort = %v, want %v", got.ServerPort, tt.want.ServerPort)
				}
				if got.Domain != tt.want.Domain {
					t.Errorf("ParseAddress() Domain = %v, want %v", got.Domain, tt.want.Domain)
				}
				if got.Entity != tt.want.Entity {
					t.Errorf("ParseAddress() Entity = %v, want %v", got.Entity, tt.want.Entity)
				}
			}
		})
	}
}

func TestAddressString(t *testing.T) {
	tests := []struct {
		name string
		addr Address
		want string
	}{
		{
			name: "empty",
			addr: Address{},
			want: "",
		},
		{
			name: "remote",
			addr: Address{
				ServerPort: "hub.example.com:8443",
				Domain:     DomainAgent,
				Entity:     "uuid-123",
			},
			want: "hub.example.com:8443/AGENT/uuid-123",
		},
		{
			name: "local",
			addr: Address{
				ServerPort: "",
				Domain:     DomainFragment,
				Entity:     "uuid-456",
			},
			want: "/FRAGMENT/uuid-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.addr.String(); got != tt.want {
				t.Errorf("Address.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddressRoundTrip(t *testing.T) {
	original := Address{
		ServerPort: "hub.example.com:8443",
		Domain:     DomainRelation,
		Entity:     "550e8400-e29b-41d4-a716-446655440000",
	}

	str := original.String()
	parsed, err := ParseAddress(str)
	if err != nil {
		t.Fatalf("ParseAddress() error = %v", err)
	}

	if parsed.ServerPort != original.ServerPort {
		t.Errorf("Roundtrip ServerPort = %v, want %v", parsed.ServerPort, original.ServerPort)
	}
	if parsed.Domain != original.Domain {
		t.Errorf("Roundtrip Domain = %v, want %v", parsed.Domain, original.Domain)
	}
	if parsed.Entity != original.Entity {
		t.Errorf("Roundtrip Entity = %v, want %v", parsed.Entity, original.Entity)
	}
}

func TestAddressValidate(t *testing.T) {
	tests := []struct {
		name    string
		addr    Address
		wantErr bool
	}{
		{
			name:    "empty is valid",
			addr:    Address{},
			wantErr: false,
		},
		{
			name: "valid remote",
			addr: Address{
				ServerPort: "hub:8080",
				Domain:     DomainAgent,
				Entity:     "uuid",
			},
			wantErr: false,
		},
		{
			name: "missing domain",
			addr: Address{
				ServerPort: "hub:8080",
				Entity:     "uuid",
			},
			wantErr: true,
		},
		{
			name: "missing entity",
			addr: Address{
				ServerPort: "hub:8080",
				Domain:     DomainAgent,
			},
			wantErr: true,
		},
		{
			name: "invalid domain",
			addr: Address{
				ServerPort: "hub:8080",
				Domain:     "INVALID",
				Entity:     "uuid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.addr.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Address.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
