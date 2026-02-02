package api

import (
	"net/http"
	"strings"
)

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth)

	// Federated entities (immutable - only Create and Read)

	// Agents
	s.router.HandleFunc("/api/v1/agents", s.routeAgents)

	// Fragments
	s.router.HandleFunc("/api/v1/fragments", s.routeFragments)

	// Relations
	s.router.HandleFunc("/api/v1/relations", s.routeRelations)
	s.router.HandleFunc("/api/v1/entities/", s.handleGetEntityRelations)

	// Tags
	s.router.HandleFunc("/api/v1/tags", s.routeTags)
	s.router.HandleFunc("/api/v1/tags/", s.routeTagsWithID)

	// Transforms
	s.router.HandleFunc("/api/v1/transforms", s.routeTransforms)

	// Auth (challenge-response)
	s.router.HandleFunc("/api/v1/auth/challenge", s.handleCreateChallenge)
	s.router.HandleFunc("/api/v1/auth/verify", s.handleVerifyChallenge)

	// Sessions
	s.router.HandleFunc("/api/v1/sessions/", s.routeSessions)

	// Projects
	s.router.HandleFunc("/api/v1/projects", s.routeProjects)
}

// Route handlers that dispatch based on method

func (s *Server) routeAgents(w http.ResponseWriter, r *http.Request) {
	// Check if there's a UUID in the path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/agents")
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodPost:
			s.handleCreateAgent(w, r)
		case http.MethodGet:
			s.handleListAgents(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Has UUID
	uuid := strings.TrimPrefix(path, "/")
	if r.Method == http.MethodGet {
		s.handleGetAgentByUUID(w, r, uuid)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) routeFragments(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/fragments")
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodPost:
			s.handleCreateFragment(w, r)
		case http.MethodGet:
			s.handleListFragments(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	uuid := strings.TrimPrefix(path, "/")
	if r.Method == http.MethodGet {
		s.handleGetFragmentByUUID(w, r, uuid)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) routeRelations(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/relations")
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodPost:
			s.handleCreateRelation(w, r)
		case http.MethodGet:
			s.handleListRelations(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	uuid := strings.TrimPrefix(path, "/")
	if r.Method == http.MethodGet {
		s.handleGetRelationByUUID(w, r, uuid)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) routeTags(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateTag(w, r)
	case http.MethodGet:
		s.handleListTags(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) routeTagsWithID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tags/")
	
	// Check for by-name route
	if strings.HasPrefix(path, "by-name/") {
		name := strings.TrimPrefix(path, "by-name/")
		if r.Method == http.MethodGet {
			s.handleGetTagByNameParam(w, r, name)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// UUID route
	if r.Method == http.MethodGet {
		s.handleGetTagByUUID(w, r, path)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) routeTransforms(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/transforms")
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodPost:
			s.handleCreateTransform(w, r)
		case http.MethodGet:
			s.handleListTransforms(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	uuid := strings.TrimPrefix(path, "/")
	if r.Method == http.MethodGet {
		s.handleGetTransformByUUID(w, r, uuid)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) routeSessions(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
	switch r.Method {
	case http.MethodGet:
		s.handleGetSessionByID(w, r, id)
	case http.MethodDelete:
		s.handleDeleteSessionByID(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) routeProjects(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/projects")
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodPost:
			s.handleCreateProject(w, r)
		case http.MethodGet:
			s.handleListProjects(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	id := strings.TrimPrefix(path, "/")
	switch r.Method {
	case http.MethodGet:
		s.handleGetProjectByID(w, r, id)
	case http.MethodPut:
		s.handleUpdateProjectByID(w, r, id)
	case http.MethodDelete:
		s.handleDeleteProjectByID(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleHealth returns the server health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "1.0.0",
	})
}
