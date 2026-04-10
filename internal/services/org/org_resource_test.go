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
	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"

	upClient "github.com/catalin4513/terraform-provider-uptrace-ce/internal/client"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

func newTestClient(t *testing.T, handler http.Handler) *upClient.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	apiClient, err := runtime.NewAPIClient(
		srv.URL,
		runtime.WithHTTPClient(&httpDoerAdapter{client: srv.Client()}),
	)
	require.NoError(t, err)

	return &upClient.Client{
		API: generated.NewClient(apiClient),
	}
}

type httpDoerAdapter struct {
	client *http.Client
}

func (a *httpDoerAdapter) Do(_ context.Context, req *http.Request) (*http.Response, error) {
	return a.client.Do(req)
}

func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func TestOrgResource_Create(t *testing.T) {
	ctx := context.Background()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/internal/v1/orgs", r.URL.Path)

		var body generated.OrgCreateRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "Acme", body.Name)

		jsonResponse(w, http.StatusOK, generated.OrgResponse{
			Org: generated.Org{ID: 42, Name: "Acme"},
		})
	}))

	orgResource := &OrgResource{client: c}
	schemaResp := &resource.SchemaResponse{}
	orgResource.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	sch := schemaResp.Schema

	planRaw := tftypes.NewValue(sch.Type().TerraformType(ctx), map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, nil),
		"name": tftypes.NewValue(tftypes.String, "Acme"),
	})
	nullRaw := tftypes.NewValue(sch.Type().TerraformType(ctx), nil)

	createReq := resource.CreateRequest{Plan: tfsdk.Plan{Raw: planRaw, Schema: sch}}
	createResp := resource.CreateResponse{State: tfsdk.State{Raw: nullRaw, Schema: sch}}

	orgResource.Create(ctx, createReq, &createResp)
	require.False(t, createResp.Diagnostics.HasError(), "diagnostics: %v", createResp.Diagnostics)

	var state orgModel
	require.False(t, createResp.State.Get(ctx, &state).HasError())
	require.Equal(t, types.StringValue("42"), state.ID)
	require.Equal(t, types.StringValue("Acme"), state.Name)
}

func TestOrgResource_Read(t *testing.T) {
	ctx := context.Background()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/internal/v1/orgs/42", r.URL.Path)

		jsonResponse(w, http.StatusOK, generated.OrgResponse{
			Org: generated.Org{ID: 42, Name: "Acme Updated"},
		})
	}))

	orgResource := &OrgResource{client: c}
	schemaResp := &resource.SchemaResponse{}
	orgResource.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	sch := schemaResp.Schema

	stateRaw := tftypes.NewValue(sch.Type().TerraformType(ctx), map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "42"),
		"name": tftypes.NewValue(tftypes.String, "Acme"),
	})

	readReq := resource.ReadRequest{State: tfsdk.State{Raw: stateRaw, Schema: sch}}
	readResp := resource.ReadResponse{State: tfsdk.State{Raw: stateRaw, Schema: sch}}

	orgResource.Read(ctx, readReq, &readResp)
	require.False(t, readResp.Diagnostics.HasError(), "diagnostics: %v", readResp.Diagnostics)

	var state orgModel
	require.False(t, readResp.State.Get(ctx, &state).HasError())
	require.Equal(t, types.StringValue("42"), state.ID)
	require.Equal(t, types.StringValue("Acme Updated"), state.Name)
}

func TestOrgResource_Read_notFound(t *testing.T) {
	ctx := context.Background()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(w, http.StatusNotFound, map[string]string{"message": "not found"})
	}))

	orgResource := &OrgResource{client: c}
	schemaResp := &resource.SchemaResponse{}
	orgResource.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	sch := schemaResp.Schema

	stateRaw := tftypes.NewValue(sch.Type().TerraformType(ctx), map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "99"),
		"name": tftypes.NewValue(tftypes.String, "Gone"),
	})

	readReq := resource.ReadRequest{State: tfsdk.State{Raw: stateRaw, Schema: sch}}
	readResp := resource.ReadResponse{State: tfsdk.State{Raw: stateRaw, Schema: sch}}

	orgResource.Read(ctx, readReq, &readResp)
	require.False(t, readResp.Diagnostics.HasError())
	require.True(t, readResp.State.Raw.IsNull(), "state should be removed on 404")
}

func TestOrgResource_Update(t *testing.T) {
	ctx := context.Background()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		require.Equal(t, "/internal/v1/orgs/42", r.URL.Path)

		var body generated.OrgUpdateRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "Renamed", body.Name)

		jsonResponse(w, http.StatusOK, generated.OrgResponse{
			Org: generated.Org{ID: 42, Name: "Renamed"},
		})
	}))

	orgResource := &OrgResource{client: c}
	schemaResp := &resource.SchemaResponse{}
	orgResource.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	sch := schemaResp.Schema

	planRaw := tftypes.NewValue(sch.Type().TerraformType(ctx), map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "42"),
		"name": tftypes.NewValue(tftypes.String, "Renamed"),
	})

	updateReq := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Raw: planRaw, Schema: sch},
		State: tfsdk.State{Raw: planRaw, Schema: sch},
	}
	updateResp := resource.UpdateResponse{State: tfsdk.State{Raw: planRaw, Schema: sch}}

	orgResource.Update(ctx, updateReq, &updateResp)
	require.False(t, updateResp.Diagnostics.HasError(), "diagnostics: %v", updateResp.Diagnostics)

	var state orgModel
	require.False(t, updateResp.State.Get(ctx, &state).HasError())
	require.Equal(t, types.StringValue("Renamed"), state.Name)
}

func TestOrgResource_Delete(t *testing.T) {
	ctx := context.Background()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		require.Equal(t, "/internal/v1/orgs/42", r.URL.Path)

		jsonResponse(w, http.StatusOK, struct{}{})
	}))

	orgResource := &OrgResource{client: c}
	schemaResp := &resource.SchemaResponse{}
	orgResource.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	sch := schemaResp.Schema

	stateRaw := tftypes.NewValue(sch.Type().TerraformType(ctx), map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "42"),
		"name": tftypes.NewValue(tftypes.String, "Acme"),
	})

	deleteReq := resource.DeleteRequest{State: tfsdk.State{Raw: stateRaw, Schema: sch}}
	deleteResp := resource.DeleteResponse{State: tfsdk.State{Raw: stateRaw, Schema: sch}}

	orgResource.Delete(ctx, deleteReq, &deleteResp)
	require.False(t, deleteResp.Diagnostics.HasError(), "diagnostics: %v", deleteResp.Diagnostics)
}
