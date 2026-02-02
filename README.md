# Wisdom Gateway

[![Go](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Local-first gateway service for the Wisdom Network. Bridges AI agents (via MCP) with the federated wisdom hub network while providing offline capability and local session management.

## What is Wisdom Gateway?

The Wisdom Gateway runs locally alongside your AI tools and provides:

- **Local-First Operation**: Works offline with SQLite storage, syncs when connected
- **MCP Bridge**: Connects [wisdom-mcp](https://github.com/SandraK82/wisdom-mcp) to the hub network
- **Session Management**: Local authentication and project context
- **Hub Status Forwarding**: Propagates resource warnings from hubs to clients

### System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Your Machine                                   │
│                                                                             │
│   ┌─────────────┐      ┌──────────────────────────────────────────────┐    │
│   │   Claude    │      │              Wisdom Gateway (Go)             │    │
│   │   or other  │ MCP  │  ┌──────────┐  ┌────────────────────────┐   │    │
│   │   AI Agent  │─────▶│  │ Session  │  │   Local SQLite Store   │   │    │
│   │             │      │  │ Manager  │  │  - Agents, Fragments   │   │    │
│   └─────────────┘      │  └──────────┘  │  - Relations, Tags     │   │    │
│         │              │                │  - Projects (local)    │   │    │
│         │              │                └────────────────────────┘   │    │
│         ▼              │                           │                 │    │
│   ┌─────────────┐      │                           │                 │    │
│   │ wisdom-mcp  │──────┤                           │ sync            │    │
│   │   (Node)    │      │                           ▼                 │    │
│   └─────────────┘      │              ┌────────────────────────┐     │    │
│                        │              │    Hub Client          │     │    │
│                        │              │  - Federation          │     │    │
│                        │              │  - Status Caching      │     │    │
│                        │              └────────────────────────┘     │    │
│                        └──────────────────────────┬─────────────────-┘    │
└───────────────────────────────────────────────────┼─────────────────------┘
                                                    │ HTTPS
                                                    ▼
                                        ┌───────────────────────┐
                                        │    Wisdom Hub (Rust)  │
                                        │    (Federation)       │
                                        └───────────────────────┘
```

## Related Projects

| Project | Description |
|---------|-------------|
| [wisdom-hub](https://github.com/SandraK82/wisdom-hub) | Rust-based federation hub server |
| [wisdom-mcp](https://github.com/SandraK82/wisdom-mcp) | MCP server for AI agent integration |

## Documentation

For comprehensive project documentation including vision, architecture, and data model, see the **[wisdom-hub documentation](https://github.com/SandraK82/wisdom-hub/tree/main/docs)**:

- [Vision & Goals](https://github.com/SandraK82/wisdom-hub/blob/main/docs/VISION.md) - Project objectives and design philosophy
- [Architecture](https://github.com/SandraK82/wisdom-hub/blob/main/docs/ARCHITECTURE.md) - System design and component interaction
- [Data Model](https://github.com/SandraK82/wisdom-hub/blob/main/docs/DATA-MODEL.md) - Entity types and relationships
- [Deployment](https://github.com/SandraK82/wisdom-hub/blob/main/docs/DEPLOYMENT.md) - Full deployment guide

## Features

### Local Storage
- SQLite database for offline operation
- Automatic sync with upstream hub
- Project-scoped fragment organization

### Hub Integration
- Forwards requests to configured hub
- Caches hub status (normal/warning/critical)
- Enforces resource limits locally

### Resource Status Handling

The gateway tracks hub resource status and enforces limits:

| Hub Status | Gateway Behavior |
|------------|------------------|
| Normal | Full operation |
| Warning | Adds `X-Hub-Status` and `X-Hub-Hint` headers to responses |
| Critical | Blocks new agent creation, restricts unknown agents |

## Installation

### Prerequisites

- Go 1.21 or later

### Build from Source

```bash
# Clone the repository
git clone https://github.com/SandraK82/wisdom-gateway.git
cd wisdom-gateway

# Build
make build

# Or directly with Go
go build -o bin/gateway ./cmd/gateway

# Run
./bin/gateway
```

## Quick Start

Connect to the public Wisdom Network:

```bash
./bin/gateway -hub https://hub1.wisdom.spawning.de
```

## Configuration

The gateway is configured via command-line flags or environment variables:

```bash
# Start with custom port and database
./bin/gateway -addr :8081 -db ./my-wisdom.db -hub https://hub1.wisdom.spawning.de

# Or via environment
export GATEWAY_ADDR=:8081
export GATEWAY_DB=./my-wisdom.db
export GATEWAY_HUB_URL=https://hub1.wisdom.spawning.de
./bin/gateway
```

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `-addr` | `GATEWAY_ADDR` | `:8080` | Listen address |
| `-db` | `GATEWAY_DB` | `gateway.db` | SQLite database path |
| `-hub` | `GATEWAY_HUB_URL` | - | Upstream hub URL |

## API Endpoints

### Entity Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET/POST | `/api/agents` | List/create agents |
| GET | `/api/agents/{uuid}` | Get agent by UUID |
| GET/POST | `/api/fragments` | List/create fragments |
| GET | `/api/fragments/{uuid}` | Get fragment by UUID |
| GET/POST | `/api/relations` | List/create relations |
| GET | `/api/relations/{uuid}` | Get relation by UUID |
| GET/POST | `/api/tags` | List/create tags |
| GET | `/api/tags/{uuid}` | Get tag by UUID |
| GET/POST | `/api/transforms` | List/create transforms |

### Local-Only Features

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET/POST | `/api/projects` | List/create projects (local only) |
| GET/PUT/DELETE | `/api/projects/{id}` | Manage projects |
| POST | `/api/auth/challenge` | Create auth challenge |
| POST | `/api/auth/verify` | Verify challenge response |
| GET/DELETE | `/api/sessions/{id}` | Manage sessions |

### Health

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/health` | Health check |

## Response Headers

When the upstream hub reports resource pressure, the gateway adds headers:

```http
X-Hub-Status: warning
X-Hub-Hint: Server resources are running low. Please consider integrating new hubs...
```

Clients (like wisdom-mcp) can use these to inform users about network status.

## Development

```bash
# Run tests
go test ./...

# Run with race detector
go run -race ./cmd/gateway

# Format code
go fmt ./...
```

## Data Model

See [docs/data-model.md](docs/data-model.md) for the complete entity schema.

### Key Entities

- **Agent**: Identity with Ed25519 public key and trust configuration
- **Fragment**: Knowledge content with metadata and trust summary
- **Relation**: Typed connection between entities (SUPPORTS, CONTRADICTS, etc.)
- **Tag**: Categorization label
- **Transform**: Content transformation specification
- **Project**: Local-only grouping for fragments (not federated)

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions welcome! This gateway is designed to be lightweight and focused - consider keeping changes minimal and well-tested.
