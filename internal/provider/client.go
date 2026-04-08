package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client provides authenticated HTTP access to the Uptrace API.
type Client struct {
	// Endpoint is the base URL including the /internal/v1 prefix.
	Endpoint string
	// Token is the bearer authentication token.
	Token string
	// ProjectID is the default project for project-scoped operations.
	ProjectID int64
	// HTTP is the underlying HTTP client.
	HTTP *http.Client
}

// doJSON sends an HTTP request with an optional JSON body and decodes the response
// into dest. It returns an error that includes "not found" for 404 responses.
func (c *Client) doJSON(ctx context.Context, method, path string, body any, dest any) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.Endpoint+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIError(resp.StatusCode, respBody)
	}

	if dest != nil {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// apiError represents an HTTP error response from the Uptrace API.
type apiError struct {
	statusCode int
	body       string
}

func newAPIError(statusCode int, body []byte) *apiError {
	return &apiError{statusCode: statusCode, body: string(body)}
}

// Error implements the error interface.
func (e *apiError) Error() string {
	return fmt.Sprintf("status %d: %s", e.statusCode, e.body)
}

// IsNotFound reports whether the error indicates a missing resource.
func (e *apiError) IsNotFound() bool {
	return e.statusCode == http.StatusNotFound || strings.Contains(e.body, "not found")
}

// isNotFound checks whether err is an API 404 error.
func isNotFound(err error) bool {
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		return apiErr.IsNotFound()
	}
	return strings.Contains(err.Error(), "not found")
}
