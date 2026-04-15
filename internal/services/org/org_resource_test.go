//go:build integration

package org

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"

	upClient "github.com/catalin4513/terraform-provider-uptrace-ce/internal/client"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/testutil"
)

func TestMain(m *testing.M) {
	testutil.LoadEnv()
	os.Exit(m.Run())
}

func TestOrgResource_CRUD(t *testing.T) {
	ctx := context.Background()
	r, sch := newOrgResource(ctx, t)

	const budget = 250.0

	nullState := tftypes.NewValue(sch.Type().TerraformType(ctx), nil)
	createPlanRaw := planValue(ctx, sch, nil, "test-org-1", budget)

	var created orgModel
	var createdAt float32

	t.Run("create org", func(t *testing.T) {
		resp := resource.CreateResponse{State: tfsdk.State{Raw: nullState, Schema: sch}}
		r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Raw: createPlanRaw, Schema: sch}}, &resp)
		require.False(t, resp.Diagnostics.HasError())

		require.False(t, resp.State.Get(ctx, &created).HasError())
		require.Equal(t, types.StringValue("test-org-1"), created.Name)

		fresh := fetchOrg(ctx, t, r.client, created.ID.ValueString())
		require.Greater(t, fresh.ID, uint64(0))
		require.Equal(t, strconv.FormatUint(fresh.ID, 10), created.ID.ValueString())
		require.Equal(t, "test-org-1", fresh.Name)
		require.NotZero(t, fresh.CreatedAt)
		require.NotZero(t, fresh.UpdatedAt)
		createdAt = *fresh.CreatedAt
	})

	t.Run("read org", func(t *testing.T) {
		stateRaw := planValue(ctx, sch, runtime.Ptr(created.ID.ValueString()), created.Name.ValueString(), budget)
		resp := resource.ReadResponse{State: tfsdk.State{Raw: stateRaw, Schema: sch}}
		r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Raw: stateRaw, Schema: sch}}, &resp)
		require.False(t, resp.Diagnostics.HasError())

		var read orgModel
		require.False(t, resp.State.Get(ctx, &read).HasError())
		require.Equal(t, created.ID, read.ID)
		require.Equal(t, types.StringValue("test-org-1"), read.Name)
	})

	t.Run("update name and budget", func(t *testing.T) {
		const newBudget = 500.0
		priorStateRaw := planValue(ctx, sch, runtime.Ptr(created.ID.ValueString()), created.Name.ValueString(), budget)
		planRaw := planValue(ctx, sch, runtime.Ptr(created.ID.ValueString()), "test-org-2", newBudget)

		resp := resource.UpdateResponse{State: tfsdk.State{Raw: planRaw, Schema: sch}}
		r.Update(ctx, resource.UpdateRequest{
			Plan:  tfsdk.Plan{Raw: planRaw, Schema: sch},
			State: tfsdk.State{Raw: priorStateRaw, Schema: sch},
		}, &resp)
		require.False(t, resp.Diagnostics.HasError())

		var updated orgModel
		require.False(t, resp.State.Get(ctx, &updated).HasError())
		require.Equal(t, types.StringValue("test-org-2"), updated.Name)
		require.Equal(t, types.Float64Value(newBudget), updated.Budget)

		fresh := fetchOrg(ctx, t, r.client, created.ID.ValueString())
		require.Equal(t, "test-org-2", fresh.Name)
		require.NotNil(t, fresh.Budget)
		require.Equal(t, newBudget, *fresh.Budget)
		require.Equal(t, createdAt, *fresh.CreatedAt)
		require.GreaterOrEqual(t, *fresh.UpdatedAt, createdAt)
	})

	t.Run("delete org", func(t *testing.T) {
		stateRaw := planValue(ctx, sch, runtime.Ptr(created.ID.ValueString()), "test-org-2", budget)
		resp := resource.DeleteResponse{State: tfsdk.State{Raw: stateRaw, Schema: sch}}
		r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Raw: stateRaw, Schema: sch}}, &resp)
		require.False(t, resp.Diagnostics.HasError())

		orgID, err := parseOrgID(created.ID.ValueString())
		require.NoError(t, err)
		_, err = r.client.API.GetOrg(ctx, &generated.GetOrgRequestOptions{
			PathParams: &generated.GetOrgPath{OrgID: orgID},
		})
		require.Error(t, err)

		created.ID = types.StringNull()
	})

	t.Cleanup(func() {
		if created.ID.IsNull() || created.ID.ValueString() == "" {
			return
		}
		if os.Getenv("KEEP_TEST_ORG") != "" {
			t.Logf("KEEP_TEST_ORG set; leaving org %s in place", created.ID.ValueString())
			return
		}
		stateRaw := planValue(ctx, sch, runtime.Ptr(created.ID.ValueString()), created.Name.ValueString(), budget)
		delResp := resource.DeleteResponse{State: tfsdk.State{Raw: stateRaw, Schema: sch}}
		r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Raw: stateRaw, Schema: sch}}, &delResp)
		if delResp.Diagnostics.HasError() {
			t.Logf("cleanup delete diagnostics: %v", delResp.Diagnostics)
		}
	})
}

func TestOrgResource_Read_notFound(t *testing.T) {
	ctx := context.Background()
	r, sch := newOrgResource(ctx, t)

	stateRaw := planValue(ctx, sch, runtime.Ptr("18446744073709551615"), "ghost", 0.0)
	readResp := resource.ReadResponse{State: tfsdk.State{Raw: stateRaw, Schema: sch}}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Raw: stateRaw, Schema: sch}}, &readResp)

	require.False(t, readResp.Diagnostics.HasError())
	require.True(t, readResp.State.Raw.IsNull())
}

func newClient(t *testing.T) *upClient.Client {
	t.Helper()

	endpoint := os.Getenv("UPTRACE_ENDPOINT")
	token := os.Getenv("UPTRACE_TOKEN")
	if endpoint == "" || token == "" {
		t.Fatal("UPTRACE_ENDPOINT and UPTRACE_TOKEN must be set")
	}

	c := upClient.New(endpoint, token, 0)
	if _, err := c.API.ListOrgs(context.Background()); err != nil {
		t.Fatalf("failed to connect to Uptrace API at %s: %v", endpoint, err)
	}
	return c
}

func newOrgResource(ctx context.Context, t *testing.T) (*OrgResource, rschema.Schema) {
	t.Helper()
	r := &OrgResource{client: newClient(t)}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	return r, schemaResp.Schema
}

func planValue(ctx context.Context, sch rschema.Schema, id *string, name string, budget float64) tftypes.Value {
	var idVal any
	if id != nil {
		idVal = *id
	}
	return tftypes.NewValue(sch.Type().TerraformType(ctx), map[string]tftypes.Value{
		"id":     tftypes.NewValue(tftypes.String, idVal),
		"name":   tftypes.NewValue(tftypes.String, name),
		"budget": tftypes.NewValue(tftypes.Number, budget),
	})
}

func fetchOrg(ctx context.Context, t *testing.T, c *upClient.Client, id string) *generated.Org {
	t.Helper()
	orgID, err := parseOrgID(id)
	require.NoError(t, err)
	out, err := c.API.GetOrg(ctx, &generated.GetOrgRequestOptions{
		PathParams: &generated.GetOrgPath{OrgID: orgID},
	})
	require.NoError(t, err)
	return &out.Org
}
