package org

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/clients"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/errs"
)

var (
	_ resource.Resource                = &OrgResource{}
	_ resource.ResourceWithConfigure   = &OrgResource{}
	_ resource.ResourceWithImportState = &OrgResource{}
)

// OrgResource manages an Uptrace organization.
type OrgResource struct {
	client *clients.Client
}

type orgModel struct {
	ID        types.String  `tfsdk:"id"`
	Name      types.String  `tfsdk:"name"`
	Budget    types.Float64 `tfsdk:"budget"`
	CreatedAt types.String  `tfsdk:"created_at"`
	UpdatedAt types.String  `tfsdk:"updated_at"`
}

type apiOrg struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Budget    float64 `json:"budget"`
	CreatedAt float64 `json:"createdAt"`
	UpdatedAt float64 `json:"updatedAt"`
}

func NewOrgResource() resource.Resource {
	return &OrgResource{}
}

// NewOrgResourceWithClient is a test helper that injects a pre-built client.
func NewOrgResourceWithClient(c *clients.Client) *OrgResource {
	return &OrgResource{client: c}
}

func (r *OrgResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org"
}

func (r *OrgResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"budget": schema.Float64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *OrgResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*clients.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected provider data type",
			fmt.Sprintf("expected *clients.Client, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *OrgResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan orgModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := map[string]any{
		"name": plan.Name.ValueString(),
	}
	if !plan.Budget.IsNull() && !plan.Budget.IsUnknown() {
		in["budget"] = plan.Budget.ValueFloat64()
	}

	tflog.Info(ctx, "creating org", map[string]any{"name": plan.Name.ValueString()})

	var out struct {
		Org apiOrg `json:"org"`
	}
	if err := r.client.DoJSON(ctx, http.MethodPost, "/orgs", in, &out); err != nil {
		resp.Diagnostics.AddError("create org failed", err.Error())
		return
	}

	apiOrgToModel(&out.Org, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state orgModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var out struct {
		Org apiOrg `json:"org"`
	}
	if err := r.client.DoJSON(ctx, http.MethodGet, "/orgs/"+state.ID.ValueString(), nil, &out); err != nil {
		if errs.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read org failed", err.Error())
		return
	}

	apiOrgToModel(&out.Org, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan orgModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := map[string]any{
		"name": plan.Name.ValueString(),
	}

	tflog.Info(ctx, "updating org", map[string]any{"id": plan.ID.ValueString()})

	var out struct {
		Org apiOrg `json:"org"`
	}
	if err := r.client.DoJSON(ctx, http.MethodPut, "/orgs/"+plan.ID.ValueString(), in, &out); err != nil {
		resp.Diagnostics.AddError("update org failed", err.Error())
		return
	}

	apiOrgToModel(&out.Org, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state orgModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "deleting org", map[string]any{"id": state.ID.ValueString()})

	err := r.client.DoJSON(ctx, http.MethodDelete, "/orgs/"+state.ID.ValueString(), nil, nil)
	if err != nil && !errs.IsNotFound(err) {
		resp.Diagnostics.AddError("delete org failed", err.Error())
	}
}

func (r *OrgResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func milliToRFC3339(ms float64) string {
	sec := int64(ms / 1000)
	nsec := int64((ms - float64(sec)*1000) * 1e6)
	return time.Unix(sec, nsec).UTC().Format(time.RFC3339)
}

func apiOrgToModel(org *apiOrg, m *orgModel) {
	m.ID = types.StringValue(strconv.FormatInt(org.ID, 10))
	m.Name = types.StringValue(org.Name)

	if org.Budget != 0 {
		m.Budget = types.Float64Value(org.Budget)
	} else {
		m.Budget = types.Float64Null()
	}

	if org.CreatedAt != 0 {
		m.CreatedAt = types.StringValue(milliToRFC3339(org.CreatedAt))
	}
	if org.UpdatedAt != 0 {
		m.UpdatedAt = types.StringValue(milliToRFC3339(org.UpdatedAt))
	}
}
