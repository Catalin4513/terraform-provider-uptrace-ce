package org

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/clients"
)

func TestOrgResource_CRUD(t *testing.T) {
	ctx := context.Background()

	type apiCall struct {
		method      string
		path        string
		auth        string
		contentType string
		body        apiOrg
	}
	var calls []apiCall

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := apiCall{
			method:      r.Method,
			path:        r.URL.Path,
			auth:        r.Header.Get("Authorization"),
			contentType: r.Header.Get("Content-Type"),
		}
		if call.contentType != "" {
			_ = json.NewDecoder(r.Body).Decode(&call.body)
		}
		calls = append(calls, call)

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/orgs":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"org":{"id":1,"name":"Acme","budget":250.5,"createdAt":1700000000000,"updatedAt":1700000060000}}`))

		case r.Method == http.MethodDelete && r.URL.Path == "/orgs/1":
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	orgResource := NewOrgResourceWithClient(&clients.Client{Endpoint: srv.URL, Token: "test-token", HTTP: srv.Client()})
	schemaResp := &resource.SchemaResponse{}
	orgResource.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	sch := schemaResp.Schema

	planRaw := tftypes.NewValue(sch.Type().TerraformType(ctx), map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, nil),
		"name":       tftypes.NewValue(tftypes.String, "Acme"),
		"budget":     tftypes.NewValue(tftypes.Number, 250.5),
		"created_at": tftypes.NewValue(tftypes.String, nil),
		"updated_at": tftypes.NewValue(tftypes.String, nil),
	})
	nullRaw := tftypes.NewValue(sch.Type().TerraformType(ctx), nil)

	createReq := resource.CreateRequest{Plan: tfsdk.Plan{Raw: planRaw, Schema: sch}}
	createResp := resource.CreateResponse{State: tfsdk.State{Raw: nullRaw, Schema: sch}}

	t.Run("create", func(t *testing.T) {
		orgResource.Create(ctx, createReq, &createResp)
		require.False(t, createResp.Diagnostics.HasError(), "diagnostics: %v", createResp.Diagnostics)

		var state orgModel
		require.False(t, createResp.State.Get(ctx, &state).HasError())

		require.Equal(t, orgModel{
			ID:        types.StringValue("1"),
			Name:      types.StringValue("Acme"),
			Budget:    types.Float64Value(250.5),
			CreatedAt: types.StringValue("2023-11-14T22:13:20Z"),
			UpdatedAt: types.StringValue("2023-11-14T22:14:20Z"),
		}, state)

		require.Equal(t, []apiCall{{
			method:      http.MethodPost,
			path:        "/orgs",
			auth:        "Bearer test-token",
			contentType: "application/json",
			body:        apiOrg{Name: "Acme", Budget: 250.5},
		}}, calls)
	})

	t.Run("delete", func(t *testing.T) {
		callsBefore := len(calls)

		deleteReq := resource.DeleteRequest{State: tfsdk.State{Raw: createResp.State.Raw, Schema: sch}}
		deleteResp := resource.DeleteResponse{State: tfsdk.State{Raw: createResp.State.Raw, Schema: sch}}

		orgResource.Delete(ctx, deleteReq, &deleteResp)
		require.False(t, deleteResp.Diagnostics.HasError(), "diagnostics: %v", deleteResp.Diagnostics)

		require.Equal(t, []apiCall{{
			method: http.MethodDelete,
			path:   "/orgs/1",
			auth:   "Bearer test-token",
		}}, calls[callsBefore:])
	})
}

func TestApiOrgToModel(t *testing.T) {
	tests := []struct {
		name string
		in   apiOrg
		want orgModel
	}{
		{
			name: "all fields populated",
			in: apiOrg{
				ID:        7,
				Name:      "Acme",
				Budget:    250.5,
				CreatedAt: 1700000000000,
				UpdatedAt: 1700000060000,
			},
			want: orgModel{
				ID:        types.StringValue("7"),
				Name:      types.StringValue("Acme"),
				Budget:    types.Float64Value(250.5),
				CreatedAt: types.StringValue("2023-11-14T22:13:20Z"),
				UpdatedAt: types.StringValue("2023-11-14T22:14:20Z"),
			},
		},
		{
			name: "zero budget becomes null",
			in: apiOrg{
				ID:     1,
				Name:   "NoBudget",
				Budget: 0,
			},
			want: orgModel{
				ID:     types.StringValue("1"),
				Name:   types.StringValue("NoBudget"),
				Budget: types.Float64Null(),
			},
		},
	}

	for _, tt := range tests {
		var got orgModel
		apiOrgToModel(&tt.in, &got)
		require.Equal(t, tt.want.ID, got.ID, tt.name)
		require.Equal(t, tt.want.Name, got.Name, tt.name)
		require.Equal(t, tt.want.Budget, got.Budget, tt.name)
		require.Equal(t, tt.want.CreatedAt, got.CreatedAt, tt.name)
		require.Equal(t, tt.want.UpdatedAt, got.UpdatedAt, tt.name)
	}
}
