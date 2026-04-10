package client

import (
	"context"
	"net/http"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

const defaultRequestTimeout = 30 * time.Second

// Client wraps the generated OpenAPI client with retry and auth configuration.
type Client struct {
	// API is the oapi-codegen generated client for all spec-defined endpoints.
	API *generated.Client
	// ProjectID is the default project for project-scoped operations.
	ProjectID int64
}

// New creates a client with retryable HTTP transport and bearer token auth.
func New(endpoint, token string, projectID int64) *Client {
	rc := retryablehttp.NewClient()
	rc.HTTPClient = &http.Client{Timeout: defaultRequestTimeout}
	rc.Logger = nil

	bearerAuth := func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}

	apiClient, _ := runtime.NewAPIClient(
		endpoint,
		runtime.WithHTTPClient(&httpDoerAdapter{client: rc.StandardClient()}),
		runtime.WithRequestEditorFn(bearerAuth),
	)

	return &Client{
		API:       generated.NewClient(apiClient),
		ProjectID: projectID,
	}
}

// httpDoerAdapter adapts a standard *http.Client to the runtime.HttpRequestDoer
// interface which expects Do(context.Context, *http.Request).
type httpDoerAdapter struct {
	client *http.Client
}

func (a *httpDoerAdapter) Do(_ context.Context, req *http.Request) (*http.Response, error) {
	return a.client.Do(req)
}
