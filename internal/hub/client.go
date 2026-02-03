// Package hub provides an HTTP client for the wisdom-hub API.
package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/thoughtracker/shared-wisdom/internal/models"
)

// Client is an HTTP client for the wisdom-hub API.
type Client struct {
	hubURL     string // Full URL, e.g. "https://hub1.wisdom.spawning.de"
	hubHost    string // host:port for address construction, e.g. "hub1.wisdom.spawning.de:443"
	httpClient *http.Client
}

// NewClient creates a new hub client.
// hubURL: full URL like "https://hub1.wisdom.spawning.de"
func NewClient(hubURL string) *Client {
	hubURL = strings.TrimRight(hubURL, "/")

	// Derive host:port from URL
	hubHost := deriveHubHost(hubURL)

	return &Client{
		hubURL:  hubURL,
		hubHost: hubHost,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HubHost returns the hub host:port string for address construction.
func (c *Client) HubHost() string {
	return c.hubHost
}

// deriveHubHost extracts host:port from a URL for address construction.
func deriveHubHost(hubURL string) string {
	// Remove protocol
	host := hubURL
	if idx := strings.Index(host, "://"); idx >= 0 {
		host = host[idx+3:]
	}
	// Remove path
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	// Add default port if not present
	if !strings.Contains(host, ":") {
		if strings.HasPrefix(hubURL, "https") {
			host += ":443"
		} else {
			host += ":80"
		}
	}
	return host
}

// hubResponse wraps the hub API response format.
type hubResponse struct {
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"data"`
	Error     string          `json:"error,omitempty"`
	HubStatus string          `json:"hub_status,omitempty"`
}

// isRetryable returns true if the error or status code warrants a retry.
func isRetryable(statusCode int, err error) bool {
	if err != nil {
		// Network errors, timeouts
		return true
	}
	// Retry on 5xx and 429 (rate limit)
	return statusCode >= 500 || statusCode == 429
}

// post sends a POST request to the hub with retry logic (max 3 attempts, exponential backoff).
func (c *Client) post(ctx context.Context, path string, body interface{}, result interface{}) error {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 500ms, 2s, 8s
			backoff := time.Duration(1<<uint(attempt-1)*500) * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		lastErr = c.doPost(ctx, path, bodyJSON, result)
		if lastErr == nil {
			return nil
		}

		// Don't retry client errors (4xx except 429) or conflicts
		var conflict *models.ConflictError
		if errors.As(lastErr, &conflict) {
			return lastErr
		}

		// Check if the error message indicates a non-retryable status
		if strings.Contains(lastErr.Error(), "hub returned status 4") && !strings.Contains(lastErr.Error(), "status 429") {
			return lastErr
		}
	}

	return fmt.Errorf("hub request failed after %d attempts: %w", maxRetries, lastErr)
}

// doPost executes a single POST request to the hub.
func (c *Client) doPost(ctx context.Context, path string, bodyJSON []byte, result interface{}) error {
	url := c.hubURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("hub request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read hub response: %w", err)
	}

	// Try to parse as hub response wrapper first
	var hubResp hubResponse
	if err := json.Unmarshal(respBody, &hubResp); err == nil && hubResp.Data != nil {
		// Hub wraps response in {success, data, error}
		if !hubResp.Success {
			return fmt.Errorf("hub error: %s", hubResp.Error)
		}
		if result != nil {
			return json.Unmarshal(hubResp.Data, result)
		}
		return nil
	}

	// Direct JSON response (no wrapper)
	if resp.StatusCode == http.StatusConflict {
		return models.NewConflictError("entity", "already exists on hub")
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("hub returned status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		return json.Unmarshal(respBody, result)
	}
	return nil
}

// PushAgent sends an agent to the hub. Returns the hub response.
func (c *Client) PushAgent(ctx context.Context, agent *models.Agent) (*models.Agent, error) {
	var result models.Agent
	err := c.post(ctx, "/api/v1/agents", agent, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// PushFragment sends a fragment to the hub. Returns the hub response.
func (c *Client) PushFragment(ctx context.Context, fragment *models.Fragment) (*models.Fragment, error) {
	var result models.Fragment
	err := c.post(ctx, "/api/v1/fragments", fragment, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// PushTag sends a tag to the hub. Returns the hub response.
func (c *Client) PushTag(ctx context.Context, tag *models.Tag) (*models.Tag, error) {
	var result models.Tag
	err := c.post(ctx, "/api/v1/tags", tag, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// PushRelation sends a relation to the hub. Returns the hub response.
func (c *Client) PushRelation(ctx context.Context, relation *models.Relation) (*models.Relation, error) {
	var result models.Relation
	err := c.post(ctx, "/api/v1/relations", relation, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// PushTransform sends a transform to the hub. Returns the hub response.
func (c *Client) PushTransform(ctx context.Context, transform *models.Transform) (*models.Transform, error) {
	var result models.Transform
	err := c.post(ctx, "/api/v1/transforms", transform, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
