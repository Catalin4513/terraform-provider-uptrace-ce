package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

const (
	defaultRequestTimeout = 30 * time.Second
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

// IsNotFound reports whether err is an API 404 error.
func IsNotFound(err error) bool {
	return hasStatusCode(err, http.StatusNotFound)
}

// IsForbidden reports whether err is an API 403 error.
func IsForbidden(err error) bool {
	return hasStatusCode(err, http.StatusForbidden)
}

func hasStatusCode(err error, code int) bool {
	clientErr, ok := errors.AsType[*runtime.ClientAPIError](err)
	if !ok {
		return false
	}
	return clientErr.StatusCode() == code
}

// httpDoerAdapter adapts a standard *http.Client to the runtime.HttpRequestDoer
// interface which expects Do(context.Context, *http.Request).
type httpDoerAdapter struct {
	client *http.Client
}

func (a *httpDoerAdapter) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return a.client.Do(req.WithContext(ctx))
}

// ParseOrgID parses a string organization ID into a uint64.
func ParseOrgID(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

// ParseProjectID parses a string project ID into a uint32.
func ParseProjectID(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// ParseTokenID parses a string token ID into a uint64.
func ParseTokenID(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

// ParseChannelID parses a string notification channel ID into an int64.
func ParseChannelID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// ParseMonitorID parses a string monitor ID into an int64.
func ParseMonitorID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// ImportStateCompoundID parses a ":"-separated compound import ID and writes
// each segment to a matching state attribute. The last field is mapped to
// "id" (Terraform primary-key convention); earlier fields are written to
// attributes whose name equals the field name.
func ImportStateCompoundID(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
	fields ...string,
) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != len(fields) {
		resp.Diagnostics.AddError(
			"invalid import ID",
			fmt.Sprintf("expected format <%s>, got %q", strings.Join(fields, ">:<"), req.ID),
		)
		return
	}
	for i, name := range fields {
		if parts[i] == "" {
			resp.Diagnostics.AddError(
				"invalid import ID",
				fmt.Sprintf("%s segment must not be empty", name),
			)
			return
		}
	}
	for i, name := range fields {
		attr := name
		if i == len(fields)-1 {
			attr = "id"
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr), parts[i])...)
	}
}

// ResourceFromProviderData unwraps the provider data passed to a Resource's
// Configure method. Returns nil during pre-configure (ProviderData is still
// nil) without emitting diagnostics; emits a diagnostic error and returns nil
// for an unexpected type.
func ResourceFromProviderData(providerData any, diags *diag.Diagnostics) *Client {
	if providerData == nil {
		return nil
	}
	c, ok := providerData.(*Client)
	if !ok {
		diags.AddError(
			"unexpected provider data type",
			fmt.Sprintf("expected *client.Client, got %T", providerData),
		)
		return nil
	}
	return c
}
