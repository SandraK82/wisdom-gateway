// Package store provides persistence for Shared-Wisdom entities.
package store

import (
	"context"

	"github.com/thoughtracker/shared-wisdom/internal/models"
)

// Store defines the persistence interface for all entities.
type Store interface {
	// Lifecycle
	Close() error

	// Agents (federated, immutable after creation)
	CreateAgent(ctx context.Context, agent *models.Agent) error
	GetAgent(ctx context.Context, uuid string) (*models.Agent, error)
	ListAgents(ctx context.Context, opts ListOptions) ([]*models.Agent, error)
	GetAgentByPublicKey(ctx context.Context, publicKey string) (*models.Agent, error)

	// Fragments (federated, immutable)
	CreateFragment(ctx context.Context, fragment *models.Fragment) error
	GetFragment(ctx context.Context, uuid string) (*models.Fragment, error)
	ListFragments(ctx context.Context, opts FragmentListOptions) ([]*models.Fragment, error)

	// Relations (federated, immutable)
	CreateRelation(ctx context.Context, relation *models.Relation) error
	GetRelation(ctx context.Context, uuid string) (*models.Relation, error)
	ListRelations(ctx context.Context, opts RelationListOptions) ([]*models.Relation, error)
	GetRelationsForEntity(ctx context.Context, entityAddr string) ([]*models.Relation, error)

	// Tags (federated, immutable)
	CreateTag(ctx context.Context, tag *models.Tag) error
	GetTag(ctx context.Context, uuid string) (*models.Tag, error)
	GetTagByName(ctx context.Context, name string) (*models.Tag, error)
	ListTags(ctx context.Context, opts TagListOptions) ([]*models.Tag, error)

	// Transforms (federated, immutable)
	CreateTransform(ctx context.Context, transform *models.Transform) error
	GetTransform(ctx context.Context, uuid string) (*models.Transform, error)
	ListTransforms(ctx context.Context, opts ListOptions) ([]*models.Transform, error)

	// Sessions (local, mutable)
	CreateSession(ctx context.Context, session *models.Session) error
	GetSession(ctx context.Context, id string) (*models.Session, error)
	UpdateSessionLastUsed(ctx context.Context, id string) error
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) (int64, error)

	// Auth Challenges (local, mutable)
	CreateAuthChallenge(ctx context.Context, challenge *models.AuthChallenge) error
	GetAuthChallenge(ctx context.Context, id string) (*models.AuthChallenge, error)
	DeleteAuthChallenge(ctx context.Context, id string) error
	DeleteExpiredChallenges(ctx context.Context) (int64, error)

	// Projects (local, mutable)
	CreateProject(ctx context.Context, project *models.Project) error
	GetProject(ctx context.Context, id string) (*models.Project, error)
	UpdateProject(ctx context.Context, project *models.Project) error
	DeleteProject(ctx context.Context, id string) error
	ListProjects(ctx context.Context, agentUUID string) ([]*models.Project, error)
}

// ListOptions contains common pagination options.
// Supports both offset-based (legacy) and cursor-based pagination.
type ListOptions struct {
	Limit  int
	Offset int    // Legacy offset-based pagination
	Cursor string // Cursor-based pagination (UUID of last item)
}

// CursorResult wraps a list result with cursor information.
type CursorResult[T any] struct {
	Items      []T
	NextCursor string
}

// FragmentListOptions contains options for listing fragments.
type FragmentListOptions struct {
	ListOptions
	CreatorUUID string   // Filter by creator
	TagUUIDs    []string // Filter by tags (AND)
}

// RelationListOptions contains options for listing relations.
type RelationListOptions struct {
	ListOptions
	Type     models.RelationType // Filter by type
	FromAddr string              // Filter by from address
	ToAddr   string              // Filter by to address
	ByAgent  string              // Filter by asserting agent
}

// TagListOptions contains options for listing tags.
type TagListOptions struct {
	ListOptions
	Category models.TagCategory // Filter by category
	Search   string             // Search in name/content
}
