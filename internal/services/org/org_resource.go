package org

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/client"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/errs"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

var (
	_ resource.Resource                = &OrgResource{}
	_ resource.ResourceWithConfigure   = &OrgResource{}
	_ resource.ResourceWithImportState = &OrgResource{}
)

// OrgResource manages an Uptrace organization.
type OrgResource struct {
	client *client.Client
}

type orgModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewOrgResource() resource.Resource {
	return &OrgResource{}
}

func (r *OrgResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org"
}

func (r *OrgResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Organization ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Organization name.",
			},
		},
	}
}

func (r *OrgResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected provider data type",
			fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *OrgResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan orgModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "creating org", map[string]any{"name": plan.Name.ValueString()})

	out, err := r.client.API.CreateOrg(ctx, &generated.CreateOrgRequestOptions{
		Body: &generated.OrgCreateRequest{
			Name: plan.Name.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("create org failed", err.Error())
		return
	}

	orgToModel(&out.Org, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state orgModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID, err := parseOrgID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid org ID", err.Error())
		return
	}

	out, err := r.client.API.GetOrg(ctx, &generated.GetOrgRequestOptions{
		PathParams: &generated.GetOrgPath{OrgID: orgID},
	})
	if err != nil {
		if errs.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read org failed", err.Error())
		return
	}

	orgToModel(&out.Org, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan orgModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID, err := parseOrgID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid org ID", err.Error())
		return
	}

	tflog.Info(ctx, "updating org", map[string]any{"id": plan.ID.ValueString()})

	out, err := r.client.API.UpdateOrg(ctx, &generated.UpdateOrgRequestOptions{
		PathParams: &generated.UpdateOrgPath{OrgID: orgID},
		Body: &generated.OrgUpdateRequest{
			Name: plan.Name.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("update org failed", err.Error())
		return
	}

	orgToModel(&out.Org, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state orgModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID, err := parseOrgID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid org ID", err.Error())
		return
	}

	tflog.Info(ctx, "deleting org", map[string]any{"id": state.ID.ValueString()})

	_, err = r.client.API.DeleteOrg(ctx, &generated.DeleteOrgRequestOptions{
		PathParams: &generated.DeleteOrgPath{OrgID: orgID},
	})
	if err != nil && !errs.IsNotFound(err) {
		resp.Diagnostics.AddError("delete org failed", err.Error())
	}
}

func (r *OrgResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func orgToModel(org *generated.Org, m *orgModel) {
	m.ID = types.StringValue(strconv.FormatUint(org.ID, 10))
	m.Name = types.StringValue(org.Name)
}

func parseOrgID(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}
