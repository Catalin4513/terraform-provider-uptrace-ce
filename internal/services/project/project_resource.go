package project

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/client"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

var (
	_ resource.Resource                = &ProjectResource{}
	_ resource.ResourceWithConfigure   = &ProjectResource{}
	_ resource.ResourceWithImportState = &ProjectResource{}
)

// ProjectResource manages an Uptrace project scoped under an organization.
type ProjectResource struct {
	client *client.Client
}

type projectModel struct {
	ID    types.String `tfsdk:"id"`
	OrgID types.String `tfsdk:"org_id"`
	Name  types.String `tfsdk:"name"`

	GroupByEnv          types.Bool `tfsdk:"group_by_env"`
	GroupFuncsByService types.Bool `tfsdk:"group_funcs_by_service"`

	SemconvVersion     types.String `tfsdk:"semconv_version"`
	DisplayLogSeverity types.Bool   `tfsdk:"display_log_severity"`
	CountDistinct      types.Bool   `tfsdk:"count_distinct"`

	SpanTimeRange  types.Float64 `tfsdk:"span_time_range"`
	LogTimeRange   types.Float64 `tfsdk:"log_time_range"`
	EventTimeRange types.Float64 `tfsdk:"event_time_range"`

	SpanRetention   types.Float64 `tfsdk:"span_retention"`
	LogRetention    types.Float64 `tfsdk:"log_retention"`
	EventRetention  types.Float64 `tfsdk:"event_retention"`
	MetricRetention types.Float64 `tfsdk:"metric_retention"`
}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

func (r *ProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	optionalComputedBool := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: desc,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		}
	}
	semconvVersionAttr := schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Semantic conventions version. One of: none, v1.25.0, v1.33.0.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
		Validators: []validator.String{
			stringvalidator.OneOf("none", "v1.25.0", "v1.33.0"),
		},
	}
	optionalFloat64 := func(desc string) schema.Float64Attribute {
		return schema.Float64Attribute{
			Optional:    true,
			Description: desc,
		}
	}
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Project ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "Organization ID that owns this project. Changing this forces recreation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Project name.",
			},
			"group_by_env":           optionalComputedBool("Group spans by deployment environment."),
			"group_funcs_by_service": optionalComputedBool("Group functions by service name."),
			"semconv_version":        semconvVersionAttr,
			"display_log_severity":   optionalComputedBool("Display log severity column."),
			"count_distinct":         optionalComputedBool("Enable count distinct aggregations."),

			"span_time_range":  optionalFloat64("Default span query time range, in milliseconds. Omit to use the server default."),
			"log_time_range":   optionalFloat64("Default log query time range, in milliseconds. Omit to use the server default."),
			"event_time_range": optionalFloat64("Default event query time range, in milliseconds. Omit to use the server default."),

			"span_retention":   optionalFloat64("Span retention duration, in milliseconds. Omit to use the server default."),
			"log_retention":    optionalFloat64("Log retention duration, in milliseconds. Omit to use the server default."),
			"event_retention":  optionalFloat64("Event retention duration, in milliseconds. Omit to use the server default."),
			"metric_retention": optionalFloat64("Metric retention duration, in milliseconds. Omit to use the server default."),
		},
	}
}

func (r *ProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID, err := parseOrgID(plan.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid org_id", err.Error())
		return
	}

	tflog.Info(ctx, "creating project", map[string]any{
		"org_id": plan.OrgID.ValueString(),
		"name":   plan.Name.ValueString(),
	})

	body := projectRequestBody(&plan)
	out, err := r.client.API.CreateProject(ctx, &generated.CreateProjectRequestOptions{
		PathParams: &generated.CreateProjectPath{OrgID: orgID},
		Body:       body,
	})
	if err != nil {
		resp.Diagnostics.AddError("create project failed", err.Error())
		return
	}

	projectToModel(&out.Project, &plan)
	// Preserve the org_id supplied in config in case the API omits it in the response.
	plan.OrgID = types.StringValue(strconv.FormatUint(orgID, 10))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := parseProjectID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid project ID", err.Error())
		return
	}

	out, err := r.client.API.GetProject(ctx, &generated.GetProjectRequestOptions{
		PathParams: &generated.GetProjectPath{ProjectID: projectID},
	})
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read project failed", err.Error())
		return
	}

	prevOrgID := state.OrgID
	projectToModel(&out.Project, &state)
	if state.OrgID.IsNull() {
		state.OrgID = prevOrgID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := parseProjectID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid project ID", err.Error())
		return
	}

	tflog.Info(ctx, "updating project", map[string]any{"id": plan.ID.ValueString()})

	body := projectRequestBody(&plan)
	out, err := r.client.API.UpdateProject(ctx, &generated.UpdateProjectRequestOptions{
		PathParams: &generated.UpdateProjectPath{ProjectID: projectID},
		Body:       body,
	})
	if err != nil {
		resp.Diagnostics.AddError("update project failed", err.Error())
		return
	}

	prevOrgID := plan.OrgID
	projectToModel(&out.Project, &plan)
	if plan.OrgID.IsNull() {
		plan.OrgID = prevOrgID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := parseProjectID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid project ID", err.Error())
		return
	}

	tflog.Info(ctx, "deleting project", map[string]any{"id": state.ID.ValueString()})

	_, err = r.client.API.DeleteProject(ctx, &generated.DeleteProjectRequestOptions{
		PathParams: &generated.DeleteProjectPath{ProjectID: projectID},
	})
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("delete project failed", err.Error())
	}
}

// ImportState accepts "<orgID>:<projectID>" so org_id is populated even when
// the API response omits orgId.
func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"invalid import ID",
			fmt.Sprintf("expected format <org_id>:<project_id>, got %q", req.ID),
		)
		return
	}
	if _, err := parseOrgID(parts[0]); err != nil {
		resp.Diagnostics.AddError("invalid org_id in import ID", err.Error())
		return
	}
	if _, err := parseProjectID(parts[1]); err != nil {
		resp.Diagnostics.AddError("invalid project_id in import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func projectRequestBody(m *projectModel) *generated.ProjectCreateRequest {
	body := &generated.ProjectCreateRequest{
		Name: m.Name.ValueString(),
	}

	if !m.GroupByEnv.IsNull() && !m.GroupByEnv.IsUnknown() {
		v := m.GroupByEnv.ValueBool()
		body.GroupByEnv = &v
	}
	if !m.GroupFuncsByService.IsNull() && !m.GroupFuncsByService.IsUnknown() {
		v := m.GroupFuncsByService.ValueBool()
		body.GroupFuncsByService = &v
	}

	if !m.SemconvVersion.IsNull() && !m.SemconvVersion.IsUnknown() {
		v := generated.ProjectCreateRequestSemconvVersion(m.SemconvVersion.ValueString())
		body.SemconvVersion = &v
	}
	if !m.DisplayLogSeverity.IsNull() && !m.DisplayLogSeverity.IsUnknown() {
		v := m.DisplayLogSeverity.ValueBool()
		body.DisplayLogSeverity = &v
	}
	if !m.CountDistinct.IsNull() && !m.CountDistinct.IsUnknown() {
		v := m.CountDistinct.ValueBool()
		body.CountDistinct = &v
	}

	if v := float64PtrIfSet(m.SpanTimeRange); v != nil {
		body.SpanTimeRange = v
	}
	if v := float64PtrIfSet(m.LogTimeRange); v != nil {
		body.LogTimeRange = v
	}
	if v := float64PtrIfSet(m.EventTimeRange); v != nil {
		body.EventTimeRange = v
	}

	if v := float64PtrIfSet(m.SpanRetention); v != nil {
		body.SpanRetention = v
	}
	if v := float64PtrIfSet(m.LogRetention); v != nil {
		body.LogRetention = v
	}
	if v := float64PtrIfSet(m.EventRetention); v != nil {
		body.EventRetention = v
	}
	if v := float64PtrIfSet(m.MetricRetention); v != nil {
		body.MetricRetention = v
	}

	return body
}

func float64PtrIfSet(v types.Float64) *float64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	f := v.ValueFloat64()
	return &f
}

func projectToModel(p *generated.Project, m *projectModel) {
	m.ID = types.StringValue(strconv.FormatInt(int64(p.ID), 10))
	m.Name = types.StringValue(p.Name)

	if p.OrgID != nil {
		m.OrgID = types.StringValue(strconv.FormatUint(*p.OrgID, 10))
	} else {
		m.OrgID = types.StringNull()
	}

	m.GroupByEnv = boolFromPtr(p.GroupByEnv)
	m.GroupFuncsByService = boolFromPtr(p.GroupFuncsByService)

	if p.SemconvVersion != nil {
		m.SemconvVersion = types.StringValue(string(*p.SemconvVersion))
	} else {
		m.SemconvVersion = types.StringNull()
	}
	m.DisplayLogSeverity = boolFromPtr(p.DisplayLogSeverity)
	m.CountDistinct = boolFromPtr(p.CountDistinct)

	// Intentionally do NOT populate SpanTimeRange/.../MetricRetention from the
	// API response. Those attributes are Optional-only in the schema (no
	// Computed), so the user's HCL is the source of truth. Leaving them
	// untouched here preserves whatever the caller already put in the model
	// and sidesteps CE's SelectByID retention override.
}

func boolFromPtr(v *bool) types.Bool {
	if v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}

func parseProjectID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func parseOrgID(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}
