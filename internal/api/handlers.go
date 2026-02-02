package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thoughtracker/shared-wisdom/internal/models"
	"github.com/thoughtracker/shared-wisdom/internal/store"
)

// Agent handlers

func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	// Check if hub is at critical level - reject new agents
	if s.hubStatus != nil && s.hubStatus.IsCritical() {
		msg := "Hub at capacity. New agents not accepted."
		if s.hubStatus.Hint != "" {
			msg += " " + s.hubStatus.Hint
		}
		writeErrorWithStatus(w, http.StatusServiceUnavailable, msg, s.hubStatus)
		return
	}

	var agent models.Agent
	if err := parseJSON(r, &agent); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if err := agent.Validate(); err != nil {
		handleError(w, err)
		return
	}

	if err := s.store.CreateAgent(r.Context(), &agent); err != nil {
		handleError(w, err)
		return
	}

	writeJSONWithStatus(w, http.StatusCreated, agent, s.hubStatus)
}

func (s *Server) handleGetAgentByUUID(w http.ResponseWriter, r *http.Request, uuid string) {
	agent, err := s.store.GetAgent(r.Context(), uuid)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	opts := store.ListOptions{
		Limit:  parseIntParam(r, "limit", 100),
		Offset: parseIntParam(r, "offset", 0),
	}

	agents, err := s.store.ListAgents(r.Context(), opts)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agents)
}

// Fragment handlers

func (s *Server) handleCreateFragment(w http.ResponseWriter, r *http.Request) {
	var fragment models.Fragment
	if err := parseJSON(r, &fragment); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	// Check if hub is at critical level - only known agents can create content
	if s.hubStatus != nil && s.hubStatus.IsCritical() {
		// Check if the creator agent is known
		agentUUID := fragment.Creator.Entity
		_, err := s.store.GetAgent(r.Context(), agentUUID)
		if err != nil {
			msg := "Hub at capacity. Unknown agents cannot create content."
			if s.hubStatus.Hint != "" {
				msg += " " + s.hubStatus.Hint
			}
			writeErrorWithStatus(w, http.StatusServiceUnavailable, msg, s.hubStatus)
			return
		}
	}

	if err := fragment.Validate(); err != nil {
		handleError(w, err)
		return
	}

	if err := s.store.CreateFragment(r.Context(), &fragment); err != nil {
		handleError(w, err)
		return
	}

	writeJSONWithStatus(w, http.StatusCreated, fragment, s.hubStatus)
}

func (s *Server) handleGetFragmentByUUID(w http.ResponseWriter, r *http.Request, uuid string) {
	fragment, err := s.store.GetFragment(r.Context(), uuid)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, fragment)
}

func (s *Server) handleListFragments(w http.ResponseWriter, r *http.Request) {
	opts := store.FragmentListOptions{
		ListOptions: store.ListOptions{
			Limit:  parseIntParam(r, "limit", 100),
			Offset: parseIntParam(r, "offset", 0),
		},
	}

	fragments, err := s.store.ListFragments(r.Context(), opts)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, fragments)
}

// Relation handlers

func (s *Server) handleCreateRelation(w http.ResponseWriter, r *http.Request) {
	var relation models.Relation
	if err := parseJSON(r, &relation); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if err := relation.Validate(); err != nil {
		handleError(w, err)
		return
	}

	if err := s.store.CreateRelation(r.Context(), &relation); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, relation)
}

func (s *Server) handleGetRelationByUUID(w http.ResponseWriter, r *http.Request, uuid string) {
	relation, err := s.store.GetRelation(r.Context(), uuid)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, relation)
}

func (s *Server) handleListRelations(w http.ResponseWriter, r *http.Request) {
	opts := store.RelationListOptions{
		ListOptions: store.ListOptions{
			Limit:  parseIntParam(r, "limit", 100),
			Offset: parseIntParam(r, "offset", 0),
		},
		Type: models.RelationType(r.URL.Query().Get("type")),
	}

	relations, err := s.store.ListRelations(r.Context(), opts)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, relations)
}

func (s *Server) handleGetEntityRelations(w http.ResponseWriter, r *http.Request) {
	// Path: /api/v1/entities/{addr}/relations
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/entities/")
	path = strings.TrimSuffix(path, "/relations")
	
	relations, err := s.store.GetRelationsForEntity(r.Context(), path)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, relations)
}

// Tag handlers

func (s *Server) handleCreateTag(w http.ResponseWriter, r *http.Request) {
	var tag models.Tag
	if err := parseJSON(r, &tag); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if err := tag.Validate(); err != nil {
		handleError(w, err)
		return
	}

	if err := s.store.CreateTag(r.Context(), &tag); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, tag)
}

func (s *Server) handleGetTagByUUID(w http.ResponseWriter, r *http.Request, uuid string) {
	tag, err := s.store.GetTag(r.Context(), uuid)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tag)
}

func (s *Server) handleGetTagByNameParam(w http.ResponseWriter, r *http.Request, name string) {
	tag, err := s.store.GetTagByName(r.Context(), name)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tag)
}

func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) {
	opts := store.TagListOptions{
		ListOptions: store.ListOptions{
			Limit:  parseIntParam(r, "limit", 100),
			Offset: parseIntParam(r, "offset", 0),
		},
		Category: models.TagCategory(r.URL.Query().Get("category")),
		Search:   r.URL.Query().Get("search"),
	}

	tags, err := s.store.ListTags(r.Context(), opts)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

// Transform handlers

func (s *Server) handleCreateTransform(w http.ResponseWriter, r *http.Request) {
	var transform models.Transform
	if err := parseJSON(r, &transform); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if err := transform.Validate(); err != nil {
		handleError(w, err)
		return
	}

	if err := s.store.CreateTransform(r.Context(), &transform); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, transform)
}

func (s *Server) handleGetTransformByUUID(w http.ResponseWriter, r *http.Request, uuid string) {
	transform, err := s.store.GetTransform(r.Context(), uuid)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, transform)
}

func (s *Server) handleListTransforms(w http.ResponseWriter, r *http.Request) {
	opts := store.ListOptions{
		Limit:  parseIntParam(r, "limit", 100),
		Offset: parseIntParam(r, "offset", 0),
	}

	transforms, err := s.store.ListTransforms(r.Context(), opts)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, transforms)
}

// Auth handlers

type challengeRequest struct {
	AgentUUID string `json:"agent_uuid"`
}

type challengeResponse struct {
	ChallengeID string `json:"challenge_id"`
	Challenge   string `json:"challenge"`
	ExpiresAt   string `json:"expires_at"`
}

func (s *Server) handleCreateChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req challengeRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if req.AgentUUID == "" {
		writeError(w, http.StatusBadRequest, "agent_uuid is required")
		return
	}

	// Verify agent exists
	_, err := s.store.GetAgent(r.Context(), req.AgentUUID)
	if err != nil {
		handleError(w, err)
		return
	}

	// Generate challenge
	challengeBytes := make([]byte, 32)
	if _, err := rand.Read(challengeBytes); err != nil {
		handleError(w, err)
		return
	}

	challenge := &models.AuthChallenge{
		ID:        uuid.New().String(),
		AgentUUID: req.AgentUUID,
		Challenge: base64.StdEncoding.EncodeToString(challengeBytes),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := s.store.CreateAuthChallenge(r.Context(), challenge); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, challengeResponse{
		ChallengeID: challenge.ID,
		Challenge:   challenge.Challenge,
		ExpiresAt:   challenge.ExpiresAt.Format(time.RFC3339),
	})
}

type verifyRequest struct {
	ChallengeID string `json:"challenge_id"`
	Signature   string `json:"signature"`
}

type verifyResponse struct {
	SessionID string `json:"session_id"`
	ExpiresAt string `json:"expires_at"`
}

func (s *Server) handleVerifyChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req verifyRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	// Get challenge
	challenge, err := s.store.GetAuthChallenge(r.Context(), req.ChallengeID)
	if err != nil {
		handleError(w, err)
		return
	}

	if challenge.IsExpired() {
		s.store.DeleteAuthChallenge(r.Context(), req.ChallengeID)
		writeError(w, http.StatusUnauthorized, "Challenge expired")
		return
	}

	// NOTE: Signature verification happens in MCP code, not here.
	// The gateway trusts that the MCP client has verified the signature.
	// This is a placeholder for the challenge-response flow.

	// Delete used challenge
	s.store.DeleteAuthChallenge(r.Context(), req.ChallengeID)

	// Create session
	session := &models.Session{
		ID:        uuid.New().String(),
		AgentUUID: challenge.AgentUUID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		LastUsed:  time.Now(),
	}

	if err := s.store.CreateSession(r.Context(), session); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, verifyResponse{
		SessionID: session.ID,
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
	})
}

// Session handlers

func (s *Server) handleGetSessionByID(w http.ResponseWriter, r *http.Request, id string) {
	session, err := s.store.GetSession(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}

	if session.IsExpired() {
		s.store.DeleteSession(r.Context(), id)
		writeError(w, http.StatusUnauthorized, "Session expired")
		return
	}

	s.store.UpdateSessionLastUsed(r.Context(), id)
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) handleDeleteSessionByID(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.store.DeleteSession(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Project handlers

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var project models.Project
	if err := parseJSON(r, &project); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	project.ID = uuid.New().String()
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()

	if err := project.Validate(); err != nil {
		handleError(w, err)
		return
	}

	if err := s.store.CreateProject(r.Context(), &project); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) handleGetProjectByID(w http.ResponseWriter, r *http.Request, id string) {
	project, err := s.store.GetProject(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) handleUpdateProjectByID(w http.ResponseWriter, r *http.Request, id string) {
	var project models.Project
	if err := parseJSON(r, &project); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	project.ID = id

	if err := s.store.UpdateProject(r.Context(), &project); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, project)
}

func (s *Server) handleDeleteProjectByID(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.store.DeleteProject(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	agentUUID := r.URL.Query().Get("agent_uuid")
	if agentUUID == "" {
		writeError(w, http.StatusBadRequest, "agent_uuid query parameter is required")
		return
	}

	projects, err := s.store.ListProjects(r.Context(), agentUUID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

// Helper functions

func parseIntParam(r *http.Request, name string, defaultVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}
