// Package api provides the HTTP API for the Shared-Wisdom gateway.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/thoughtracker/shared-wisdom/internal/hub"
	"github.com/thoughtracker/shared-wisdom/internal/models"
	"github.com/thoughtracker/shared-wisdom/internal/store"
)

// Server is the HTTP API server for the gateway.
type Server struct {
	store     store.Store
	router    *http.ServeMux
	server    *http.Server
	hubStatus *models.HubStatus // Cached status from the upstream hub
	hubMu     sync.RWMutex      // Protects hubStatus
	syncer    *hub.Syncer       // Hub syncer (nil if no hub configured)
}

// Config holds server configuration.
type Config struct {
	Addr   string // Listen address (e.g., ":8080")
	HubURL string // Upstream hub URL
}

// NewServer creates a new API server.
func NewServer(s store.Store, cfg Config) *Server {
	srv := &Server{
		store:  s,
		router: http.NewServeMux(),
	}

	// Initialize hub syncer if hub URL is configured
	if cfg.HubURL != "" {
		hubClient := hub.NewClient(cfg.HubURL)
		srv.syncer = hub.NewSyncer(hubClient, s)
		log.Printf("Hub sync enabled: %s (host: %s)", cfg.HubURL, hubClient.HubHost())
	}

	srv.setupRoutes()

	srv.server = &http.Server{
		Addr:         cfg.Addr,
		Handler:      srv.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return srv
}

// Start starts the server.
func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.server.Addr)
	go s.cleanupExpiredEntries()
	return s.server.ListenAndServe()
}

// cleanupExpiredEntries periodically removes expired sessions and challenges.
func (s *Server) cleanupExpiredEntries() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		ctx := context.Background()
		if n, err := s.store.DeleteExpiredSessions(ctx); err != nil {
			log.Printf("Error cleaning expired sessions: %v", err)
		} else if n > 0 {
			log.Printf("Cleaned %d expired sessions", n)
		}
		if n, err := s.store.DeleteExpiredChallenges(ctx); err != nil {
			log.Printf("Error cleaning expired challenges: %v", err)
		} else if n > 0 {
			log.Printf("Cleaned %d expired challenges", n)
		}
	}
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Response helpers

type apiError struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// CursorListResponse is the paginated response format for list endpoints.
// This matches the wisdom-hub API format.
type CursorListResponse struct {
	Items      interface{} `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	writeJSONWithStatus(w, status, data, nil)
}

func writeJSONWithStatus(w http.ResponseWriter, status int, data interface{}, hubStatus *models.HubStatus) {
	w.Header().Set("Content-Type", "application/json")

	// Add hub status headers if not normal
	if hubStatus != nil && !hubStatus.IsNormal() {
		w.Header().Set("X-Hub-Status", string(hubStatus.Level))
		if hubStatus.Hint != "" {
			w.Header().Set("X-Hub-Hint", hubStatus.Hint)
		}
	}

	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, apiError{Error: message})
}

func writeErrorWithStatus(w http.ResponseWriter, status int, message string, hubStatus *models.HubStatus) {
	writeJSONWithStatus(w, status, apiError{Error: message}, hubStatus)
}

func handleError(w http.ResponseWriter, err error) {
	var notFound *models.NotFoundError
	var conflict *models.ConflictError
	var validation *models.ValidationError

	switch {
	case errors.As(err, &notFound):
		writeJSON(w, http.StatusNotFound, apiError{
			Error: err.Error(),
			Code:  "NOT_FOUND",
		})
	case errors.As(err, &conflict):
		writeJSON(w, http.StatusConflict, apiError{
			Error: err.Error(),
			Code:  "CONFLICT",
		})
	case errors.As(err, &validation):
		writeJSON(w, http.StatusBadRequest, apiError{
			Error: err.Error(),
			Code:  "VALIDATION_ERROR",
		})
	default:
		log.Printf("Internal error: %v", err)
		writeJSON(w, http.StatusInternalServerError, apiError{
			Error: "Internal server error",
			Code:  "INTERNAL_ERROR",
		})
	}
}

func parseJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	return json.NewDecoder(r.Body).Decode(v)
}

// SetHubStatus updates the cached hub status
func (s *Server) SetHubStatus(status *models.HubStatus) {
	s.hubMu.Lock()
	defer s.hubMu.Unlock()
	s.hubStatus = status
}

// GetHubStatus returns the cached hub status
func (s *Server) GetHubStatus() *models.HubStatus {
	s.hubMu.RLock()
	defer s.hubMu.RUnlock()
	return s.hubStatus
}

// UpdateHubStatusFromResponse extracts hub status from an HTTP response
func (s *Server) UpdateHubStatusFromResponse(headers http.Header) {
	s.hubMu.Lock()
	defer s.hubMu.Unlock()

	statusHeader := headers.Get("X-Hub-Status")
	if statusHeader == "" {
		s.hubStatus = nil
		return
	}

	s.hubStatus = &models.HubStatus{
		Level: models.ResourceLevel(statusHeader),
		Hint:  headers.Get("X-Hub-Hint"),
	}
}
