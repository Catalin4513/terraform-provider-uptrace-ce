package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

const (
	defaultRequestTimeout = 10 * time.Second
	maxRetries            = 2
	retryWaitMax          = 5 * time.Second
)

// Client wraps the generated OpenAPI client with retry and auth configuration.
type Client struct {
	// API is the oapi-codegen generated client for all spec-defined endpoints.
	API *generated.Client
}

// New creates a client with retryable HTTP transport and bearer token auth.
func New(endpoint, token string) (*Client, error) {
	rc := retryablehttp.NewClient()
	rc.HTTPClient = &http.Client{Timeout: defaultRequestTimeout}
	rc.RetryMax = maxRetries
	rc.RetryWaitMax = retryWaitMax
	rc.Logger = nil

	bearerAuth := func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}

	apiClient, err := runtime.NewAPIClient(
		endpoint,
		runtime.WithHTTPClient(&httpDoerAdapter{client: rc.StandardClient()}),
		runtime.WithRequestEditorFn(bearerAuth),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		API: generated.NewClient(apiClient),
	}, nil
}

// IsNotFound reports whether err indicates the resource does not exist.
func IsNotFound(err error) bool {
	clientErr, ok := errors.AsType[*runtime.ClientAPIError](err)
	if !ok {
		return false
	}
	code := clientErr.StatusCode()
	return code == http.StatusNotFound || code == http.StatusForbidden
}

// httpDoerAdapter adapts a standard *http.Client to the runtime.HttpRequestDoer
// interface which expects Do(context.Context, *http.Request).
type httpDoerAdapter struct {
	client *http.Client
}

func (a *httpDoerAdapter) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return a.client.Do(req.WithContext(ctx))
}
