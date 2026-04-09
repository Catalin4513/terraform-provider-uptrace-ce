// Package errs holds error types and classifiers shared across the provider.
package errs

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIError represents a non-2xx HTTP response from the Uptrace API.
type APIError struct {
	StatusCode int
	Body       string
}

func NewAPIError(statusCode int, body []byte) *APIError {
	return &APIError{StatusCode: statusCode, Body: string(body)}
}

func (e *APIError) Error() string {
	return fmt.Sprintf("status %d: %s", e.StatusCode, e.Body)
}

func (e *APIError) isNotFound() bool {
	return e.StatusCode == http.StatusNotFound || strings.Contains(e.Body, "not found")
}

// IsNotFound reports whether err is an API 404 (or 404-like) error.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.isNotFound()
	}
	return err != nil && strings.Contains(err.Error(), "not found")
}
