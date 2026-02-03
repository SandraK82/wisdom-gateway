package hub

import (
	"context"
	"errors"
	"log"

	"github.com/thoughtracker/shared-wisdom/internal/models"
	"github.com/thoughtracker/shared-wisdom/internal/store"
)

// Syncer orchestrates synchronization between the gateway and the hub.
type Syncer struct {
	client *Client
	store  store.Store
}

// NewSyncer creates a new syncer.
func NewSyncer(client *Client, store store.Store) *Syncer {
	return &Syncer{
		client: client,
		store:  store,
	}
}

// Client returns the underlying hub client.
func (s *Syncer) Client() *Client {
	return s.client
}

// ShouldSync returns true if the project is public and should be synced.
func (s *Syncer) ShouldSync(ctx context.Context, projectUUID string) bool {
	if projectUUID == "" {
		// No project specified — default to sync
		return true
	}

	project, err := s.store.GetProject(ctx, projectUUID)
	if err != nil {
		// If project not found, default to sync
		log.Printf("Warning: could not get project %s for visibility check: %v", projectUUID, err)
		return true
	}

	return project.Visibility == models.VisibilityPublic || project.Visibility == ""
}

// EnsureAgent ensures the agent is registered on the hub.
// This is idempotent — 409 Conflict is treated as success.
func (s *Syncer) EnsureAgent(ctx context.Context, agentUUID string) error {
	agent, err := s.store.GetAgent(ctx, agentUUID)
	if err != nil {
		return err
	}

	_, err = s.client.PushAgent(ctx, agent)
	if err != nil {
		var conflict *models.ConflictError
		if errors.As(err, &conflict) {
			// Agent already exists on hub — that's fine
			return nil
		}
		return err
	}

	return nil
}

// SyncTag syncs a tag to the hub. Tags are always synced (infrastructure).
// This is idempotent — 409 Conflict is treated as success.
func (s *Syncer) SyncTag(ctx context.Context, tag *models.Tag) (*models.Tag, error) {
	hubTag, err := s.client.PushTag(ctx, tag)
	if err != nil {
		var conflict *models.ConflictError
		if errors.As(err, &conflict) {
			// Tag already exists on hub — return the local version
			return tag, nil
		}
		return nil, err
	}
	return hubTag, nil
}
