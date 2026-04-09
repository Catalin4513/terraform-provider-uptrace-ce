package clients

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/require"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/errs"
)

func TestNew_retriesTransientFailures(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"org":{"id":1}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token", 0)
	// Reach into the retryablehttp settings to make backoff fast for tests.
	rt, ok := c.HTTP.Transport.(*retryablehttp.RoundTripper)
	require.True(t, ok, "expected retryablehttp transport")
	rt.Client.RetryWaitMin = 1 * time.Millisecond
	rt.Client.RetryWaitMax = 5 * time.Millisecond

	var out struct {
		Org struct {
			ID int64 `json:"id"`
		} `json:"org"`
	}
	err := c.DoJSON(context.Background(), http.MethodGet, "/orgs/1", nil, &out)

	require.NoError(t, err)
	require.Equal(t, int32(3), atomic.LoadInt32(&attempts), "expected 3 attempts (2 failures + 1 success)")
	require.Equal(t, int64(1), out.Org.ID)
}

func TestClient_DoJSON(t *testing.T) {
	type capture struct {
		method      string
		path        string
		auth        string
		contentType string
		body        map[string]any
	}

	tests := []struct {
		name         string
		method       string
		path         string
		in           any
		respStatus   int
		respBody     string
		wantErr      bool
		wantNotFound bool
		wantErrMsg   string
		check        func(t *testing.T, got capture, outID int64)
	}{
		{
			name:       "GET decodes response and sets bearer auth",
			method:     http.MethodGet,
			path:       "/orgs/42",
			respStatus: http.StatusOK,
			respBody:   `{"org":{"id":42}}`,
			check: func(t *testing.T, got capture, outID int64) {
				require.Equal(t, http.MethodGet, got.method)
				require.Equal(t, "/orgs/42", got.path)
				require.Equal(t, "Bearer test-token", got.auth)
				require.Empty(t, got.contentType, "GET must not send Content-Type")
				require.Equal(t, int64(42), outID)
			},
		},
		{
			name:       "POST sends JSON body and Content-Type",
			method:     http.MethodPost,
			path:       "/orgs",
			in:         map[string]any{"name": "Acme", "budget": 100.0},
			respStatus: http.StatusCreated,
			respBody:   `{"org":{"id":1}}`,
			check: func(t *testing.T, got capture, _ int64) {
				require.Equal(t, "application/json", got.contentType)
				require.Equal(t, "Acme", got.body["name"])
				require.Equal(t, 100.0, got.body["budget"])
			},
		},
		{
			name:         "404 status is classified as not-found",
			method:       http.MethodGet,
			path:         "/orgs/1",
			respStatus:   http.StatusNotFound,
			respBody:     `{"error":"missing"}`,
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name:         "non-404 with not-found body is classified as not-found",
			method:       http.MethodGet,
			path:         "/orgs/1",
			respStatus:   http.StatusInternalServerError,
			respBody:     `{"error":"org not found"}`,
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name:       "500 without not-found phrase is an error but not not-found",
			method:     http.MethodGet,
			path:       "/orgs/1",
			respStatus: http.StatusInternalServerError,
			respBody:   `{"error":"boom"}`,
			wantErr:    true,
		},
		{
			name:       "malformed JSON surfaces a decode error",
			method:     http.MethodGet,
			path:       "/orgs/1",
			respStatus: http.StatusOK,
			respBody:   `not json`,
			wantErr:    true,
			wantErrMsg: "decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got capture
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got.method = r.Method
				got.path = r.URL.Path
				got.auth = r.Header.Get("Authorization")
				got.contentType = r.Header.Get("Content-Type")
				if got.contentType != "" {
					require.NoError(t, json.NewDecoder(r.Body).Decode(&got.body))
				} else {
					_, _ = io.ReadAll(r.Body)
				}
				w.WriteHeader(tt.respStatus)
				_, _ = w.Write([]byte(tt.respBody))
			}))
			defer srv.Close()

			c := &Client{Endpoint: srv.URL, Token: "test-token", HTTP: srv.Client()}

			var out struct {
				Org struct {
					ID int64 `json:"id"`
				} `json:"org"`
			}
			err := c.DoJSON(context.Background(), tt.method, tt.path, tt.in, &out)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					require.Contains(t, err.Error(), tt.wantErrMsg)
				}
				require.Equal(t, tt.wantNotFound, errs.IsNotFound(err))
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, got, out.Org.ID)
			}
		})
	}
}
