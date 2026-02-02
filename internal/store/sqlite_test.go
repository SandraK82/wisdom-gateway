package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/thoughtracker/shared-wisdom/internal/models"
)

func setupTestStore(t *testing.T) (*SQLiteStore, func()) {
	t.Helper()
	
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to create store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.Remove(tmpFile.Name())
	}

	return store, cleanup
}

func TestAgentCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	// Create agent
	agent := &models.Agent{
		UUID:        "agent-uuid-1",
		PublicKey:   "test-public-key",
		Version:     1,
		Description: "Test agent",
		Trust: models.TrustStore{
			NumTrusts: 1,
			Trusts: []models.Trust{
				{
					Agent: models.Address{
						ServerPort: "other-hub:8080",
						Domain:     models.DomainAgent,
						Entity:     "other-agent",
					},
					Trust: 0.8,
				},
			},
		},
		Signature: "test-signature",
	}

	if err := store.CreateAgent(ctx, agent); err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}

	// Get agent
	got, err := store.GetAgent(ctx, agent.UUID)
	if err != nil {
		t.Fatalf("GetAgent() error = %v", err)
	}

	if got.UUID != agent.UUID {
		t.Errorf("GetAgent() UUID = %v, want %v", got.UUID, agent.UUID)
	}
	if got.PublicKey != agent.PublicKey {
		t.Errorf("GetAgent() PublicKey = %v, want %v", got.PublicKey, agent.PublicKey)
	}
	if len(got.Trust.Trusts) != 1 {
		t.Errorf("GetAgent() Trust.Trusts length = %v, want 1", len(got.Trust.Trusts))
	}

	// Get by public key
	got2, err := store.GetAgentByPublicKey(ctx, agent.PublicKey)
	if err != nil {
		t.Fatalf("GetAgentByPublicKey() error = %v", err)
	}
	if got2.UUID != agent.UUID {
		t.Errorf("GetAgentByPublicKey() UUID = %v, want %v", got2.UUID, agent.UUID)
	}

	// List agents
	agents, err := store.ListAgents(ctx, ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("ListAgents() error = %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("ListAgents() length = %v, want 1", len(agents))
	}

	// Duplicate should fail
	err = store.CreateAgent(ctx, agent)
	if err == nil {
		t.Error("CreateAgent() should fail on duplicate")
	}
}

func TestFragmentCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	fragment := &models.Fragment{
		UUID: "fragment-uuid-1",
		Tags: []models.Address{
			{Domain: models.DomainTag, Entity: "tag-1"},
		},
		Content: "Test content",
		Creator: models.Address{
			ServerPort: "hub:8080",
			Domain:     models.DomainAgent,
			Entity:     "agent-1",
		},
		When:      time.Now().Truncate(time.Second),
		Signature: "test-signature",
	}

	if err := store.CreateFragment(ctx, fragment); err != nil {
		t.Fatalf("CreateFragment() error = %v", err)
	}

	got, err := store.GetFragment(ctx, fragment.UUID)
	if err != nil {
		t.Fatalf("GetFragment() error = %v", err)
	}

	if got.Content != fragment.Content {
		t.Errorf("GetFragment() Content = %v, want %v", got.Content, fragment.Content)
	}
	if len(got.Tags) != 1 {
		t.Errorf("GetFragment() Tags length = %v, want 1", len(got.Tags))
	}
}

func TestRelationCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	relation := &models.Relation{
		UUID: "relation-uuid-1",
		From: models.Address{
			Domain: models.DomainFragment,
			Entity: "fragment-1",
		},
		To: models.Address{
			Domain: models.DomainFragment,
			Entity: "fragment-2",
		},
		By: models.Address{
			Domain: models.DomainAgent,
			Entity: "agent-1",
		},
		Type:    models.RelationSupports,
		Content: "This supports that",
		Creator: models.Address{
			Domain: models.DomainAgent,
			Entity: "agent-1",
		},
		When:      time.Now().Truncate(time.Second),
		Signature: "test-signature",
	}

	if err := store.CreateRelation(ctx, relation); err != nil {
		t.Fatalf("CreateRelation() error = %v", err)
	}

	got, err := store.GetRelation(ctx, relation.UUID)
	if err != nil {
		t.Fatalf("GetRelation() error = %v", err)
	}

	if got.Type != relation.Type {
		t.Errorf("GetRelation() Type = %v, want %v", got.Type, relation.Type)
	}

	// Test self-referencing relation (typing)
	typingRelation := &models.Relation{
		UUID: "relation-uuid-2",
		From: models.Address{
			Domain: models.DomainFragment,
			Entity: "fragment-1",
		},
		By: models.Address{
			Domain: models.DomainAgent,
			Entity: "agent-1",
		},
		Type: models.RelationQuestion,
		Creator: models.Address{
			Domain: models.DomainAgent,
			Entity: "agent-1",
		},
		When:      time.Now().Truncate(time.Second),
		Signature: "test-signature",
	}

	if err := store.CreateRelation(ctx, typingRelation); err != nil {
		t.Fatalf("CreateRelation(typing) error = %v", err)
	}

	// List by type
	relations, err := store.ListRelations(ctx, RelationListOptions{
		Type: models.RelationQuestion,
	})
	if err != nil {
		t.Fatalf("ListRelations() error = %v", err)
	}
	if len(relations) != 1 {
		t.Errorf("ListRelations() length = %v, want 1", len(relations))
	}
}

func TestTagCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	tag := &models.Tag{
		UUID:     "tag-uuid-1",
		Name:     "go",
		Content:  "Go programming language",
		Version:  1,
		Category: models.TagCategoryLanguage,
		Creator: models.Address{
			Domain: models.DomainAgent,
			Entity: "agent-1",
		},
		Signature: "test-signature",
	}

	if err := store.CreateTag(ctx, tag); err != nil {
		t.Fatalf("CreateTag() error = %v", err)
	}

	got, err := store.GetTag(ctx, tag.UUID)
	if err != nil {
		t.Fatalf("GetTag() error = %v", err)
	}
	if got.Name != tag.Name {
		t.Errorf("GetTag() Name = %v, want %v", got.Name, tag.Name)
	}

	// Get by name
	got2, err := store.GetTagByName(ctx, tag.Name)
	if err != nil {
		t.Fatalf("GetTagByName() error = %v", err)
	}
	if got2.UUID != tag.UUID {
		t.Errorf("GetTagByName() UUID = %v, want %v", got2.UUID, tag.UUID)
	}

	// List by category
	tags, err := store.ListTags(ctx, TagListOptions{
		Category: models.TagCategoryLanguage,
	})
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if len(tags) != 1 {
		t.Errorf("ListTags() length = %v, want 1", len(tags))
	}
}

func TestTransformCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	transform := &models.Transform{
		UUID:          "transform-uuid-1",
		Name:          "markdown-to-html",
		Description:   "Converts Markdown to HTML",
		TransformTo:   "text/html",
		TransformFrom: "text/markdown",
		Agent: models.Address{
			Domain: models.DomainAgent,
			Entity: "agent-1",
		},
		Version:   1,
		Signature: "test-signature",
	}

	if err := store.CreateTransform(ctx, transform); err != nil {
		t.Fatalf("CreateTransform() error = %v", err)
	}

	got, err := store.GetTransform(ctx, transform.UUID)
	if err != nil {
		t.Fatalf("GetTransform() error = %v", err)
	}
	if got.Name != transform.Name {
		t.Errorf("GetTransform() Name = %v, want %v", got.Name, transform.Name)
	}
}

func TestSessionCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	session := &models.Session{
		ID:        "session-1",
		AgentUUID: "agent-1",
		CreatedAt: time.Now().Truncate(time.Second),
		ExpiresAt: time.Now().Add(24 * time.Hour).Truncate(time.Second),
		LastUsed:  time.Now().Truncate(time.Second),
	}

	if err := store.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	got, err := store.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.AgentUUID != session.AgentUUID {
		t.Errorf("GetSession() AgentUUID = %v, want %v", got.AgentUUID, session.AgentUUID)
	}

	// Delete
	if err := store.DeleteSession(ctx, session.ID); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	_, err = store.GetSession(ctx, session.ID)
	if err == nil {
		t.Error("GetSession() should fail after delete")
	}
}

func TestProjectCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	project := &models.Project{
		ID:          "project-1",
		Name:        "Test Project",
		Description: "A test project",
		AgentUUID:   "agent-1",
		Tags:        []string{"tag-1", "tag-2"},
		CreatedAt:   time.Now().Truncate(time.Second),
		UpdatedAt:   time.Now().Truncate(time.Second),
	}

	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	got, err := store.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if got.Name != project.Name {
		t.Errorf("GetProject() Name = %v, want %v", got.Name, project.Name)
	}

	// Update
	project.Name = "Updated Project"
	if err := store.UpdateProject(ctx, project); err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	got2, err := store.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() after update error = %v", err)
	}
	if got2.Name != "Updated Project" {
		t.Errorf("GetProject() Name after update = %v, want 'Updated Project'", got2.Name)
	}

	// List
	projects, err := store.ListProjects(ctx, project.AgentUUID)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("ListProjects() length = %v, want 1", len(projects))
	}

	// Delete
	if err := store.DeleteProject(ctx, project.ID); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
}
