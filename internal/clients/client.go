package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/errs"
)

const defaultRequestTimeout = 30 * time.Second

// Client provides authenticated HTTP access to the Uptrace API.
type Client struct {
	// Endpoint is the base URL including the /internal/v1 prefix.
	Endpoint string
	Token    string
	// ProjectID is the default project for project-scoped operations.
	ProjectID int64
	// HTTP is the underlying HTTP client. Production code should build a
	// Client via New so HTTP gets retry + per-attempt timeout. Tests may
	// inject a bare *http.Client to bypass retries.
	HTTP *http.Client
}

// New builds a Client whose HTTP transport applies a per-attempt timeout
// and retries network errors, 429s, and 5xx responses using retryablehttp
// defaults.
func New(endpoint, token string, projectID int64) *Client {
	rc := retryablehttp.NewClient()
	rc.HTTPClient = &http.Client{Timeout: defaultRequestTimeout}
	rc.Logger = nil

	return &Client{
		Endpoint:  endpoint,
		Token:     token,
		ProjectID: projectID,
		HTTP:      rc.StandardClient(),
	}
}

// DoJSON sends an HTTP request with an optional JSON body and decodes the
// response into dest. Non-2xx responses return an *errs.APIError.
func (c *Client) DoJSON(ctx context.Context, method, path string, body any, dest any) error {
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
		return errs.NewAPIError(resp.StatusCode, respBody)
	}

	if dest != nil {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
