package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/thoughtracker/shared-wisdom/internal/models"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed store.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// migrate creates the database schema.
func (s *SQLiteStore) migrate() error {
	schema := `
	-- Agents with embedded trust
	CREATE TABLE IF NOT EXISTS agents (
		uuid TEXT PRIMARY KEY,
		public_key TEXT NOT NULL UNIQUE,
		version INTEGER NOT NULL DEFAULT 1,
		description TEXT,
		trust_json TEXT,
		primary_hub_json TEXT,
		signature TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Fragments (minimal)
	CREATE TABLE IF NOT EXISTS fragments (
		uuid TEXT PRIMARY KEY,
		tags_json TEXT,
		transform_json TEXT,
		content TEXT NOT NULL,
		content_hash TEXT,
		creator_json TEXT NOT NULL,
		version INTEGER NOT NULL DEFAULT 1,
		when_created TIMESTAMP NOT NULL,
		signature TEXT NOT NULL,
		confidence REAL DEFAULT 0.5,
		evidence_type TEXT DEFAULT 'unknown',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Relations
	CREATE TABLE IF NOT EXISTS relations (
		uuid TEXT PRIMARY KEY,
		from_json TEXT NOT NULL,
		to_json TEXT,
		by_json TEXT NOT NULL,
		type TEXT NOT NULL,
		content TEXT,
		creator_json TEXT NOT NULL,
		version INTEGER NOT NULL DEFAULT 1,
		when_created TIMESTAMP NOT NULL,
		signature TEXT NOT NULL,
		confidence REAL DEFAULT 0.5,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_relations_type ON relations(type);
	CREATE INDEX IF NOT EXISTS idx_relations_from ON relations(from_json);

	-- Tags
	CREATE TABLE IF NOT EXISTS tags (
		uuid TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		content TEXT,
		version INTEGER NOT NULL DEFAULT 1,
		category TEXT NOT NULL,
		creator_json TEXT NOT NULL,
		signature TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name ON tags(name);

	-- Transforms
	CREATE TABLE IF NOT EXISTS transforms (
		uuid TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		tags_json TEXT,
		transform_to TEXT NOT NULL,
		transform_from TEXT NOT NULL,
		additional_data TEXT,
		agent_json TEXT NOT NULL,
		version INTEGER NOT NULL DEFAULT 1,
		signature TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Sessions (local)
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		agent_uuid TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		last_used TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_agent ON sessions(agent_uuid);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

	-- Auth Challenges (local)
	CREATE TABLE IF NOT EXISTS auth_challenges (
		id TEXT PRIMARY KEY,
		agent_uuid TEXT NOT NULL,
		challenge TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		expires_at TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_challenges_expires ON auth_challenges(expires_at);

	-- Projects (local)
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		agent_uuid TEXT NOT NULL,
		tags_json TEXT,
		visibility TEXT NOT NULL DEFAULT 'public',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_projects_agent ON projects(agent_uuid);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return err
	}

	// Migrations for existing databases
	migrations := []string{
		// Add visibility column to projects if it doesn't exist
		`ALTER TABLE projects ADD COLUMN visibility TEXT NOT NULL DEFAULT 'public'`,
	}
	for _, m := range migrations {
		// Ignore errors from already-applied migrations (e.g., "duplicate column")
		s.db.Exec(m)
	}

	return nil
}

// Agent methods

func (s *SQLiteStore) CreateAgent(ctx context.Context, agent *models.Agent) error {
	trustJSON, err := json.Marshal(agent.Trust)
	if err != nil {
		return fmt.Errorf("failed to marshal trust: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO agents (uuid, public_key, version, description, trust_json, primary_hub_json, signature)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, agent.UUID, agent.PublicKey, agent.Version, agent.Description, string(trustJSON), agent.PrimaryHub, agent.Signature)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return models.NewConflictError("agent", "agent already exists")
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) GetAgent(ctx context.Context, uuid string) (*models.Agent, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT uuid, public_key, version, description, trust_json, primary_hub_json, signature
		FROM agents WHERE uuid = ?
	`, uuid)

	return s.scanAgent(row)
}

func (s *SQLiteStore) GetAgentByPublicKey(ctx context.Context, publicKey string) (*models.Agent, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT uuid, public_key, version, description, trust_json, primary_hub_json, signature
		FROM agents WHERE public_key = ?
	`, publicKey)

	return s.scanAgent(row)
}

func (s *SQLiteStore) ListAgents(ctx context.Context, opts ListOptions) ([]*models.Agent, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT uuid, public_key, version, description, trust_json, primary_hub_json, signature
		FROM agents LIMIT ? OFFSET ?
	`, limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		agent, err := s.scanAgentRow(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

func (s *SQLiteStore) scanAgent(row *sql.Row) (*models.Agent, error) {
	var agent models.Agent
	var trustJSON sql.NullString
	var hubStr sql.NullString

	err := row.Scan(&agent.UUID, &agent.PublicKey, &agent.Version, &agent.Description,
		&trustJSON, &hubStr, &agent.Signature)
	if err == sql.ErrNoRows {
		return nil, models.NewNotFoundError("agent", "")
	}
	if err != nil {
		return nil, err
	}

	if trustJSON.Valid {
		if err := json.Unmarshal([]byte(trustJSON.String), &agent.Trust); err != nil {
			return nil, fmt.Errorf("failed to unmarshal trust: %w", err)
		}
	}

	if hubStr.Valid {
		agent.PrimaryHub = hubStr.String
	}

	return &agent, nil
}

func (s *SQLiteStore) scanAgentRow(rows *sql.Rows) (*models.Agent, error) {
	var agent models.Agent
	var trustJSON sql.NullString
	var hubStr sql.NullString

	err := rows.Scan(&agent.UUID, &agent.PublicKey, &agent.Version, &agent.Description,
		&trustJSON, &hubStr, &agent.Signature)
	if err != nil {
		return nil, err
	}

	if trustJSON.Valid {
		if err := json.Unmarshal([]byte(trustJSON.String), &agent.Trust); err != nil {
			return nil, fmt.Errorf("failed to unmarshal trust: %w", err)
		}
	}

	if hubStr.Valid {
		agent.PrimaryHub = hubStr.String
	}

	return &agent, nil
}

// Fragment methods

func (s *SQLiteStore) CreateFragment(ctx context.Context, fragment *models.Fragment) error {
	tagsJSON, err := json.Marshal(fragment.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	transformJSON, err := json.Marshal(fragment.Transform)
	if err != nil {
		return fmt.Errorf("failed to marshal transform: %w", err)
	}

	creatorJSON, err := json.Marshal(fragment.Creator)
	if err != nil {
		return fmt.Errorf("failed to marshal creator: %w", err)
	}

	// Set defaults for version and timestamps
	version := fragment.Version
	if version == 0 {
		version = 1
	}
	now := time.Now()

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO fragments (uuid, tags_json, transform_json, content, content_hash, creator_json, version, when_created, signature, confidence, evidence_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, fragment.UUID, string(tagsJSON), string(transformJSON), fragment.Content,
		fragment.ContentHash, string(creatorJSON), version, fragment.When, fragment.Signature,
		fragment.Confidence, fragment.EvidenceType, now, now)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return models.NewConflictError("fragment", "fragment already exists")
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) GetFragment(ctx context.Context, uuid string) (*models.Fragment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT uuid, tags_json, transform_json, content, content_hash, creator_json, version, when_created, signature, confidence, evidence_type, created_at, updated_at
		FROM fragments WHERE uuid = ?
	`, uuid)

	var fragment models.Fragment
	var tagsJSON, transformJSON, creatorJSON string
	var contentHash, evidenceType sql.NullString
	var whenCreated, createdAt, updatedAt time.Time

	err := row.Scan(&fragment.UUID, &tagsJSON, &transformJSON, &fragment.Content,
		&contentHash, &creatorJSON, &fragment.Version, &whenCreated, &fragment.Signature,
		&fragment.Confidence, &evidenceType, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, models.NewNotFoundError("fragment", uuid)
	}
	if err != nil {
		return nil, err
	}

	fragment.When = whenCreated
	fragment.CreatedAt = createdAt
	fragment.UpdatedAt = updatedAt
	if contentHash.Valid {
		fragment.ContentHash = contentHash.String
	}
	if evidenceType.Valid {
		fragment.EvidenceType = models.EvidenceType(evidenceType.String)
	}

	if err := json.Unmarshal([]byte(tagsJSON), &fragment.Tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}
	if err := json.Unmarshal([]byte(transformJSON), &fragment.Transform); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transform: %w", err)
	}
	if err := json.Unmarshal([]byte(creatorJSON), &fragment.Creator); err != nil {
		return nil, fmt.Errorf("failed to unmarshal creator: %w", err)
	}

	return &fragment, nil
}

func (s *SQLiteStore) ListFragments(ctx context.Context, opts FragmentListOptions) ([]*models.Fragment, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	// Simple implementation - can be extended with filters
	rows, err := s.db.QueryContext(ctx, `
		SELECT uuid, tags_json, transform_json, content, content_hash, creator_json, version, when_created, signature, confidence, evidence_type, created_at, updated_at
		FROM fragments ORDER BY when_created DESC LIMIT ? OFFSET ?
	`, limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fragments []*models.Fragment
	for rows.Next() {
		var fragment models.Fragment
		var tagsJSON, transformJSON, creatorJSON string
		var contentHash, evidenceType sql.NullString
		var whenCreated, createdAt, updatedAt time.Time

		err := rows.Scan(&fragment.UUID, &tagsJSON, &transformJSON, &fragment.Content,
			&contentHash, &creatorJSON, &fragment.Version, &whenCreated, &fragment.Signature,
			&fragment.Confidence, &evidenceType, &createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}

		fragment.When = whenCreated
		fragment.CreatedAt = createdAt
		fragment.UpdatedAt = updatedAt
		if contentHash.Valid {
			fragment.ContentHash = contentHash.String
		}
		if evidenceType.Valid {
			fragment.EvidenceType = models.EvidenceType(evidenceType.String)
		}

		if err := json.Unmarshal([]byte(tagsJSON), &fragment.Tags); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(transformJSON), &fragment.Transform); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(creatorJSON), &fragment.Creator); err != nil {
			return nil, err
		}

		fragments = append(fragments, &fragment)
	}
	return fragments, rows.Err()
}

// Relation methods

func (s *SQLiteStore) CreateRelation(ctx context.Context, relation *models.Relation) error {
	fromJSON, err := json.Marshal(relation.From)
	if err != nil {
		return fmt.Errorf("failed to marshal from: %w", err)
	}

	toJSON, err := json.Marshal(relation.To)
	if err != nil {
		return fmt.Errorf("failed to marshal to: %w", err)
	}

	byJSON, err := json.Marshal(relation.By)
	if err != nil {
		return fmt.Errorf("failed to marshal by: %w", err)
	}

	creatorJSON, err := json.Marshal(relation.Creator)
	if err != nil {
		return fmt.Errorf("failed to marshal creator: %w", err)
	}

	// Set defaults for version and timestamps
	version := relation.Version
	if version == 0 {
		version = 1
	}
	now := time.Now()

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO relations (uuid, from_json, to_json, by_json, type, content, creator_json, version, when_created, signature, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, relation.UUID, string(fromJSON), string(toJSON), string(byJSON),
		relation.Type, relation.Content, string(creatorJSON), version, relation.When, relation.Signature,
		relation.Confidence, now)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return models.NewConflictError("relation", "relation already exists")
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) GetRelation(ctx context.Context, uuid string) (*models.Relation, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT uuid, from_json, to_json, by_json, type, content, creator_json, version, when_created, signature, confidence, created_at
		FROM relations WHERE uuid = ?
	`, uuid)

	return s.scanRelation(row)
}

func (s *SQLiteStore) scanRelation(row *sql.Row) (*models.Relation, error) {
	var relation models.Relation
	var fromJSON, toJSON, byJSON, creatorJSON string
	var content sql.NullString
	var whenCreated, createdAt time.Time

	err := row.Scan(&relation.UUID, &fromJSON, &toJSON, &byJSON,
		&relation.Type, &content, &creatorJSON, &relation.Version, &whenCreated, &relation.Signature,
		&relation.Confidence, &createdAt)
	if err == sql.ErrNoRows {
		return nil, models.NewNotFoundError("relation", "")
	}
	if err != nil {
		return nil, err
	}

	relation.When = whenCreated
	relation.CreatedAt = createdAt
	if content.Valid {
		relation.Content = content.String
	}

	if err := json.Unmarshal([]byte(fromJSON), &relation.From); err != nil {
		return nil, fmt.Errorf("failed to unmarshal from: %w", err)
	}
	if err := json.Unmarshal([]byte(toJSON), &relation.To); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to: %w", err)
	}
	if err := json.Unmarshal([]byte(byJSON), &relation.By); err != nil {
		return nil, fmt.Errorf("failed to unmarshal by: %w", err)
	}
	if err := json.Unmarshal([]byte(creatorJSON), &relation.Creator); err != nil {
		return nil, fmt.Errorf("failed to unmarshal creator: %w", err)
	}

	return &relation, nil
}

func (s *SQLiteStore) ListRelations(ctx context.Context, opts RelationListOptions) ([]*models.Relation, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT uuid, from_json, to_json, by_json, type, content, creator_json, version, when_created, signature, confidence, created_at FROM relations`
	var conditions []string
	var args []interface{}

	if opts.Type != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, opts.Type)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY when_created DESC LIMIT ? OFFSET ?"
	args = append(args, limit, opts.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []*models.Relation
	for rows.Next() {
		var relation models.Relation
		var fromJSON, toJSON, byJSON, creatorJSON string
		var content sql.NullString
		var whenCreated, createdAt time.Time

		err := rows.Scan(&relation.UUID, &fromJSON, &toJSON, &byJSON,
			&relation.Type, &content, &creatorJSON, &relation.Version, &whenCreated, &relation.Signature,
			&relation.Confidence, &createdAt)
		if err != nil {
			return nil, err
		}

		relation.When = whenCreated
		relation.CreatedAt = createdAt
		if content.Valid {
			relation.Content = content.String
		}

		json.Unmarshal([]byte(fromJSON), &relation.From)
		json.Unmarshal([]byte(toJSON), &relation.To)
		json.Unmarshal([]byte(byJSON), &relation.By)
		json.Unmarshal([]byte(creatorJSON), &relation.Creator)

		relations = append(relations, &relation)
	}
	return relations, rows.Err()
}

func (s *SQLiteStore) GetRelationsForEntity(ctx context.Context, entityAddr string) ([]*models.Relation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT uuid, from_json, to_json, by_json, type, content, creator_json, version, when_created, signature, confidence, created_at
		FROM relations
		WHERE from_json LIKE ? OR to_json LIKE ?
		ORDER BY when_created DESC
	`, "%"+entityAddr+"%", "%"+entityAddr+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []*models.Relation
	for rows.Next() {
		var relation models.Relation
		var fromJSON, toJSON, byJSON, creatorJSON string
		var content sql.NullString
		var whenCreated, createdAt time.Time

		err := rows.Scan(&relation.UUID, &fromJSON, &toJSON, &byJSON,
			&relation.Type, &content, &creatorJSON, &relation.Version, &whenCreated, &relation.Signature,
			&relation.Confidence, &createdAt)
		if err != nil {
			return nil, err
		}

		relation.When = whenCreated
		relation.CreatedAt = createdAt
		if content.Valid {
			relation.Content = content.String
		}

		json.Unmarshal([]byte(fromJSON), &relation.From)
		json.Unmarshal([]byte(toJSON), &relation.To)
		json.Unmarshal([]byte(byJSON), &relation.By)
		json.Unmarshal([]byte(creatorJSON), &relation.Creator)

		relations = append(relations, &relation)
	}
	return relations, rows.Err()
}

// Tag methods

func (s *SQLiteStore) CreateTag(ctx context.Context, tag *models.Tag) error {
	creatorJSON, err := json.Marshal(tag.Creator)
	if err != nil {
		return fmt.Errorf("failed to marshal creator: %w", err)
	}

	// Set defaults
	version := tag.Version
	if version == 0 {
		version = 1
	}
	now := time.Now()

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO tags (uuid, name, content, version, category, creator_json, signature, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, tag.UUID, tag.Name, tag.Content, version, tag.Category, string(creatorJSON), tag.Signature, now)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return models.NewConflictError("tag", "tag already exists")
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) GetTag(ctx context.Context, uuid string) (*models.Tag, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT uuid, name, content, version, category, creator_json, signature, created_at
		FROM tags WHERE uuid = ?
	`, uuid)

	return s.scanTag(row)
}

func (s *SQLiteStore) GetTagByName(ctx context.Context, name string) (*models.Tag, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT uuid, name, content, version, category, creator_json, signature, created_at
		FROM tags WHERE name = ?
	`, name)

	return s.scanTag(row)
}

func (s *SQLiteStore) scanTag(row *sql.Row) (*models.Tag, error) {
	var tag models.Tag
	var creatorJSON string
	var content sql.NullString
	var createdAt time.Time

	err := row.Scan(&tag.UUID, &tag.Name, &content, &tag.Version, &tag.Category, &creatorJSON, &tag.Signature, &createdAt)
	if err == sql.ErrNoRows {
		return nil, models.NewNotFoundError("tag", "")
	}
	if err != nil {
		return nil, err
	}

	tag.CreatedAt = createdAt
	if content.Valid {
		tag.Content = content.String
	}

	if err := json.Unmarshal([]byte(creatorJSON), &tag.Creator); err != nil {
		return nil, fmt.Errorf("failed to unmarshal creator: %w", err)
	}

	return &tag, nil
}

func (s *SQLiteStore) ListTags(ctx context.Context, opts TagListOptions) ([]*models.Tag, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT uuid, name, content, version, category, creator_json, signature, created_at FROM tags`
	var conditions []string
	var args []interface{}

	if opts.Category != "" {
		conditions = append(conditions, "category = ?")
		args = append(args, opts.Category)
	}

	if opts.Search != "" {
		conditions = append(conditions, "(name LIKE ? OR content LIKE ?)")
		args = append(args, "%"+opts.Search+"%", "%"+opts.Search+"%")
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name LIMIT ? OFFSET ?"
	args = append(args, limit, opts.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		var tag models.Tag
		var creatorJSON string
		var content sql.NullString
		var createdAt time.Time

		err := rows.Scan(&tag.UUID, &tag.Name, &content, &tag.Version, &tag.Category, &creatorJSON, &tag.Signature, &createdAt)
		if err != nil {
			return nil, err
		}

		tag.CreatedAt = createdAt
		if content.Valid {
			tag.Content = content.String
		}

		json.Unmarshal([]byte(creatorJSON), &tag.Creator)
		tags = append(tags, &tag)
	}
	return tags, rows.Err()
}

// Transform methods

func (s *SQLiteStore) CreateTransform(ctx context.Context, transform *models.Transform) error {
	tagsJSON, err := json.Marshal(transform.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	agentJSON, err := json.Marshal(transform.Agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}

	// Set defaults
	version := transform.Version
	if version == 0 {
		version = 1
	}
	now := time.Now()

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO transforms (uuid, name, description, tags_json, transform_to, transform_from, additional_data, agent_json, version, signature, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, transform.UUID, transform.Name, transform.Description, string(tagsJSON),
		transform.TransformTo, transform.TransformFrom, transform.AdditionalData,
		string(agentJSON), version, transform.Signature, now)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return models.NewConflictError("transform", "transform already exists")
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) GetTransform(ctx context.Context, uuid string) (*models.Transform, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT uuid, name, description, tags_json, transform_to, transform_from, additional_data, agent_json, version, signature, created_at
		FROM transforms WHERE uuid = ?
	`, uuid)

	var transform models.Transform
	var tagsJSON, agentJSON string
	var description, additionalData sql.NullString
	var createdAt time.Time

	err := row.Scan(&transform.UUID, &transform.Name, &description, &tagsJSON,
		&transform.TransformTo, &transform.TransformFrom, &additionalData,
		&agentJSON, &transform.Version, &transform.Signature, &createdAt)
	if err == sql.ErrNoRows {
		return nil, models.NewNotFoundError("transform", uuid)
	}
	if err != nil {
		return nil, err
	}

	transform.CreatedAt = createdAt
	if description.Valid {
		transform.Description = description.String
	}
	if additionalData.Valid {
		transform.AdditionalData = additionalData.String
	}

	if err := json.Unmarshal([]byte(tagsJSON), &transform.Tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}
	if err := json.Unmarshal([]byte(agentJSON), &transform.Agent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent: %w", err)
	}

	return &transform, nil
}

func (s *SQLiteStore) ListTransforms(ctx context.Context, opts ListOptions) ([]*models.Transform, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT uuid, name, description, tags_json, transform_to, transform_from, additional_data, agent_json, version, signature, created_at
		FROM transforms ORDER BY name LIMIT ? OFFSET ?
	`, limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transforms []*models.Transform
	for rows.Next() {
		var transform models.Transform
		var tagsJSON, agentJSON string
		var description, additionalData sql.NullString
		var createdAt time.Time

		err := rows.Scan(&transform.UUID, &transform.Name, &description, &tagsJSON,
			&transform.TransformTo, &transform.TransformFrom, &additionalData,
			&agentJSON, &transform.Version, &transform.Signature, &createdAt)
		if err != nil {
			return nil, err
		}

		transform.CreatedAt = createdAt
		if description.Valid {
			transform.Description = description.String
		}
		if additionalData.Valid {
			transform.AdditionalData = additionalData.String
		}

		json.Unmarshal([]byte(tagsJSON), &transform.Tags)
		json.Unmarshal([]byte(agentJSON), &transform.Agent)
		transforms = append(transforms, &transform)
	}
	return transforms, rows.Err()
}

// Session methods (local)

func (s *SQLiteStore) CreateSession(ctx context.Context, session *models.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, agent_uuid, created_at, expires_at, last_used)
		VALUES (?, ?, ?, ?, ?)
	`, session.ID, session.AgentUUID, session.CreatedAt, session.ExpiresAt, session.LastUsed)
	return err
}

func (s *SQLiteStore) GetSession(ctx context.Context, id string) (*models.Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, agent_uuid, created_at, expires_at, last_used
		FROM sessions WHERE id = ?
	`, id)

	var session models.Session
	err := row.Scan(&session.ID, &session.AgentUUID, &session.CreatedAt, &session.ExpiresAt, &session.LastUsed)
	if err == sql.ErrNoRows {
		return nil, models.NewNotFoundError("session", id)
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *SQLiteStore) UpdateSessionLastUsed(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE sessions SET last_used = ? WHERE id = ?
	`, time.Now(), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return models.NewNotFoundError("session", id)
	}
	return nil
}

func (s *SQLiteStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Auth Challenge methods (local)

func (s *SQLiteStore) CreateAuthChallenge(ctx context.Context, challenge *models.AuthChallenge) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_challenges (id, agent_uuid, challenge, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, challenge.ID, challenge.AgentUUID, challenge.Challenge, challenge.CreatedAt, challenge.ExpiresAt)
	return err
}

func (s *SQLiteStore) GetAuthChallenge(ctx context.Context, id string) (*models.AuthChallenge, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, agent_uuid, challenge, created_at, expires_at
		FROM auth_challenges WHERE id = ?
	`, id)

	var challenge models.AuthChallenge
	err := row.Scan(&challenge.ID, &challenge.AgentUUID, &challenge.Challenge, &challenge.CreatedAt, &challenge.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, models.NewNotFoundError("auth_challenge", id)
	}
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (s *SQLiteStore) DeleteAuthChallenge(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_challenges WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) DeleteExpiredChallenges(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM auth_challenges WHERE expires_at < ?`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Project methods (local)

func (s *SQLiteStore) CreateProject(ctx context.Context, project *models.Project) error {
	tagsJSON, err := json.Marshal(project.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	visibility := project.Visibility
	if visibility == "" {
		visibility = models.VisibilityPublic
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO projects (id, name, description, agent_uuid, tags_json, visibility, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, project.ID, project.Name, project.Description, project.AgentUUID, string(tagsJSON), string(visibility), project.CreatedAt, project.UpdatedAt)
	return err
}

func (s *SQLiteStore) GetProject(ctx context.Context, id string) (*models.Project, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, agent_uuid, tags_json, visibility, created_at, updated_at
		FROM projects WHERE id = ?
	`, id)

	var project models.Project
	var tagsJSON string
	var description sql.NullString
	var visibility string

	err := row.Scan(&project.ID, &project.Name, &description, &project.AgentUUID, &tagsJSON, &visibility, &project.CreatedAt, &project.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, models.NewNotFoundError("project", id)
	}
	if err != nil {
		return nil, err
	}

	if description.Valid {
		project.Description = description.String
	}
	project.Visibility = models.Visibility(visibility)

	if err := json.Unmarshal([]byte(tagsJSON), &project.Tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	return &project, nil
}

func (s *SQLiteStore) UpdateProject(ctx context.Context, project *models.Project) error {
	tagsJSON, err := json.Marshal(project.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	project.UpdatedAt = time.Now()

	visibility := project.Visibility
	if visibility == "" {
		visibility = models.VisibilityPublic
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE projects SET name = ?, description = ?, tags_json = ?, visibility = ?, updated_at = ?
		WHERE id = ?
	`, project.Name, project.Description, string(tagsJSON), string(visibility), project.UpdatedAt, project.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return models.NewNotFoundError("project", project.ID)
	}
	return nil
}

func (s *SQLiteStore) DeleteProject(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) ListProjects(ctx context.Context, agentUUID string) ([]*models.Project, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, agent_uuid, tags_json, visibility, created_at, updated_at
		FROM projects WHERE agent_uuid = ? ORDER BY name
	`, agentUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		var project models.Project
		var tagsJSON string
		var description sql.NullString
		var visibility string

		err := rows.Scan(&project.ID, &project.Name, &description, &project.AgentUUID, &tagsJSON, &visibility, &project.CreatedAt, &project.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			project.Description = description.String
		}
		project.Visibility = models.Visibility(visibility)

		json.Unmarshal([]byte(tagsJSON), &project.Tags)
		projects = append(projects, &project)
	}
	return projects, rows.Err()
}
