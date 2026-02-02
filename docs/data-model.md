# Shared-Wisdom Data Model

This document describes the simplified data model used by the Shared-Wisdom Gateway.

## Design Philosophy

The data model follows these principles:

1. **Minimal Structures**: Each entity contains only essential fields
2. **Relations Over Enums**: Instead of type/state enums, use Relations
3. **Embedded Trust**: TrustStore embedded in Agent for efficient path-finding
4. **Federated Addressing**: Every entity has a globally unique Address

## Core Entities

### Address

A federated identifier for any entity in the network.

```go
type Address struct {
    ServerPort string        // "hub.example.com:8443" or "" for local
    Domain     AddressDomain // AGENT, TAG, FRAGMENT, RELATION, TRANSFORMATION, HUB
    Entity     string        // UUID
}
```

**Format**: `server:port/DOMAIN/entity-uuid`

**Examples**:
- Remote: `hub.example.com:8443/AGENT/550e8400-e29b-41d4-a716-446655440000`
- Local: `/FRAGMENT/abc123`

**Domains**:
| Domain | Description |
|--------|-------------|
| `AGENT` | A participant in the network |
| `TAG` | A classification label |
| `FRAGMENT` | A piece of knowledge |
| `RELATION` | A relationship between entities |
| `TRANSFORMATION` | A content transformation definition |
| `HUB` | A gateway/hub in the network |

---

### Agent

A participant in the Shared-Wisdom network.

```go
type Agent struct {
    UUID        string     // Unique identifier
    PublicKey   string     // Base64-encoded public key
    Version     uint32     // Incremented on updates
    Description string     // Human-readable description
    Trust       TrustStore // Embedded trust relationships
    PrimaryHub  Address    // Where agent's data primarily lives
    Signature   string     // Signature over agent data
}

type TrustStore struct {
    NumTrusts uint64  // Total count
    Trusts    []Trust // Direct trust relationships
}

type Trust struct {
    Agent Address // Agent being trusted/distrusted
    Trust float32 // -1.0 (distrust) to 1.0 (full trust)
}
```

**Why embedded TrustStore?**
- Enables efficient trust path calculations
- No need for separate trust table queries
- Trust relationships travel with the Agent

---

### Fragment

A piece of knowledge in the network. Fragments are **minimal** - typing and state are expressed through Relations.

```go
type Fragment struct {
    UUID      string    // Unique identifier
    Tags      []Address // References to Tag entities
    Transform Address   // Reference to Transform (how to interpret content)
    Content   string    // The actual content
    Creator   Address   // Agent who created this
    When      time.Time // Creation timestamp
    Signature string    // Signature over fragment data
}
```

**What's NOT in Fragment:**
- ❌ `Type` - Use Relations (QUESTION, HYPOTHESE, etc.)
- ❌ `State` - Use Relations or let it be implicit
- ❌ `Title` - Content is self-describing

---

### Relation

A relationship between entities. Relations serve multiple purposes:

1. **Content Relationships**: How fragments relate to each other
2. **Trust Expression**: Agents expressing trust in entities
3. **Fragment Typing**: Marking a fragment as a question, hypothesis, etc.

```go
type Relation struct {
    UUID      string       // Unique identifier
    From      Address      // Source entity (required)
    To        Address      // Target entity (optional for self-reference)
    By        Address      // Agent asserting this relation
    Type      RelationType // Type of relationship
    Content   string       // Reasoning/explanation (optional)
    Creator   Address      // Agent who created this
    When      time.Time    // Creation timestamp
    Signature string       // Signature over relation data
}
```

**Relation Types:**

| Category | Type | Description |
|----------|------|-------------|
| **Trust** | `TRUST` | Agent trusts/distrusts the From entity |
| **Content** | `SUPPORTS` | From supports To |
| | `CONTRADICTS` | From contradicts To |
| | `EXTENDS` | From extends To |
| | `SUPERSEDES` | From supersedes To |
| | `DERIVED_FROM` | From is derived from To |
| | `RELATED_TO` | Generic relation |
| | `EXAMPLE_OF` | From is an example of To |
| **Typing** | `QUESTION` | Fragment is a question |
| | `HYPOTHESE` | Fragment is a hypothesis |
| | `ANTITHESE` | Fragment is an antithesis |
| | `SYNTHESE` | Fragment is a synthesis |
| **Refinement** | `SPECIALIZES` | From specializes To |
| | `CLARIFIES` | From clarifies To |
| | `GENERALIZES` | From generalizes To |

**Self-Referencing Relations (Typing):**

To mark a Fragment as a question:
```json
{
  "from": "/FRAGMENT/abc123",
  "to": {},
  "by": "/AGENT/agent-1",
  "type": "QUESTION"
}
```

---

### Tag

A classification label for fragments.

```go
type Tag struct {
    UUID      string      // Unique identifier
    Name      string      // Unique, normalized name
    Content   string      // Description
    Version   uint32      // Incremented on updates
    Category  TagCategory // Classification category
    Creator   Address     // Agent who created this
    Signature string      // Signature over tag data
}
```

**Tag Categories:**
| Category | Examples |
|----------|----------|
| `PLATFORM` | macos, linux, windows |
| `LANGUAGE` | go, python, rust |
| `FRAMEWORK` | react, django, rails |
| `LIBRARY` | lodash, numpy |
| `VERSION` | v1.0, 2024-Q1 |
| `DOMAIN` | finance, healthcare |
| `TYPE` | tutorial, reference |
| `ENVIRONMENT` | production, development |
| `ARCHITECTURE` | microservices, monolith |
| `COUNTRY` | de, us, jp |
| `FIELD` | physics, biology |

---

### Transform

Defines how content is transformed between formats.

```go
type Transform struct {
    UUID           string    // Unique identifier
    Name           string    // Human-readable name
    Description    string    // What this transform does
    Tags           []Address // Related tags
    TransformTo    string    // Target format (e.g., "text/html")
    TransformFrom  string    // Source format (e.g., "text/markdown")
    AdditionalData string    // JSON with extra configuration
    Agent          Address   // Agent who created this
    Version        uint32    // Incremented on updates
    Signature      string    // Signature over transform data
}
```

---

## Local Entities (Gateway-specific)

These entities exist only on the local gateway and are **mutable**.

### Session

```go
type Session struct {
    ID        string    // Session UUID
    AgentUUID string    // Authenticated agent
    CreatedAt time.Time
    ExpiresAt time.Time
    LastUsed  time.Time
}
```

### AuthChallenge

```go
type AuthChallenge struct {
    ID        string    // Challenge UUID
    AgentUUID string    // Agent being challenged
    Challenge string    // Random challenge string
    CreatedAt time.Time
    ExpiresAt time.Time
}
```

### Project

```go
type Project struct {
    ID          string    // Project UUID
    Name        string    // Human-readable name
    Description string
    AgentUUID   string    // Owner agent
    Tags        []string  // Default tag UUIDs
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

---

## Removed Structures

The following structures from older designs have been **removed**:

| Structure | Replacement |
|-----------|-------------|
| `FragmentType` enum | Relations (QUESTION, HYPOTHESE, etc.) |
| `FragmentState` enum | Relations or implicit |
| `TrustRelation` table | Embedded in Agent.Trust |
| `TrustVote` | TRUST Relations |
| `AgentReputation` | Calculated from TrustStore |
| `AgentIdentity` / `AgentLocalAuth` | Simplified to Agent |
| `TagAlias` | Normalization is sufficient |
| `FilterResult`, `StructuredTag` | Not needed |

---

## Entity Relationships

```
┌─────────┐
│  Agent  │◄──────────────────────────────┐
├─────────┤                               │
│ Trust[] │──────────▶ Agent              │
└────┬────┘                               │
     │ creates                            │
     ▼                                    │
┌──────────┐      ┌──────────┐           │
│ Fragment │◄────▶│ Relation │───────────┤
├──────────┤      ├──────────┤           │
│ Tags[]   │──┐   │ From     │           │
│ Transform│  │   │ To       │           │
└──────────┘  │   │ By       │───────────┘
              │   └──────────┘
              │
              ▼
         ┌─────────┐
         │   Tag   │
         └─────────┘
              ▲
              │
         ┌────┴─────┐
         │Transform │
         └──────────┘
```

---

## Trust Model

### Two Levels of Trust

1. **TrustStore (in Agent)**: Direct trust to other agents
   - Used for path-finding in the trust network
   - Embedded for efficiency
   
2. **TRUST Relation**: Trust/distrust attached to any entity
   - Trust from known agent → increases value for me + my network
   - Distrust → decreases value
   - Unknown agent → neutral

### Trust Calculation

```
effective_trust(entity) = Σ (direct_trust[agent] × agent_trust_in_entity)
```

Where:
- `direct_trust[agent]` comes from my TrustStore
- `agent_trust_in_entity` comes from TRUST Relations by that agent
