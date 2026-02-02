# Shared-Wisdom Diagrams

This directory contains PlantUML diagrams documenting the Shared-Wisdom architecture and workflows.

## Generating Images

To generate PNG/SVG images from the PlantUML files:

```bash
# Using PlantUML JAR
java -jar plantuml.jar *.puml

# Using Docker
docker run -v $(pwd):/data plantuml/plantuml *.puml

# Using online service
# Upload to https://www.plantuml.com/plantuml/
```

## Diagrams

### Class Diagram

- **class-diagram.puml** - Complete data model showing all entities and relationships

### Sequence Diagrams

| Diagram | Description |
|---------|-------------|
| sequence-auth.puml | Authentication flow (challenge-response) |
| sequence-agents.puml | Agent registration and trust management |
| sequence-fragments.puml | Fragment creation, typing, and queries |
| sequence-relations.puml | Content, trust, and typing relations |
| sequence-tags.puml | Tag management and usage |
| sequence-trust.puml | Two-level trust model explanation |
| sequence-transforms.puml | Transform management |
| sequence-projects.puml | Local project management |

## Key Concepts Illustrated

### Immutability
Federated entities (Agent, Fragment, Relation, Tag, Transform) are immutable. The diagrams show that PUT/DELETE operations return 404 for these entities.

### Typing via Relations
Fragments don't have a `type` field. Instead, self-referencing relations (QUESTION, HYPOTHESE, etc.) express fragment types.

### Two-Level Trust
1. **TrustStore** (embedded in Agent) - for trust path finding
2. **TRUST Relations** - for entity-level trust expression

### Client-Side Crypto
The Gateway does not perform cryptographic operations. All signing and verification happens in the MCP Client.

## Updating Diagrams

When updating the data model:
1. Update `class-diagram.puml` first
2. Update relevant sequence diagrams
3. Regenerate images
4. Update documentation if needed
