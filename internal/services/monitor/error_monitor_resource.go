package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/client"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

var (
	_ resource.Resource                = &ErrorMonitorResource{}
	_ resource.ResourceWithConfigure   = &ErrorMonitorResource{}
	_ resource.ResourceWithImportState = &ErrorMonitorResource{}
)

var (
	trendAggFuncs      = []string{"sum", "avg", "median", "last"}
	trendSensitivities = []string{"low", "medium", "high"}
)

const (
	trendAggFuncDefault     = "sum"
	trendSensitivityDefault = "medium"
)

type ErrorMonitorResource struct {
	client *client.Client
}

func NewErrorMonitorResource() resource.Resource {
	return &ErrorMonitorResource{}
}

type errorMonitorModel struct {
	ID                    types.String             `tfsdk:"id"`
	ProjectID             types.String             `tfsdk:"project_id"`
	Name                  types.String             `tfsdk:"name"`
	NotifyEveryoneByEmail types.Bool               `tfsdk:"notify_everyone_by_email"`
	TrendAggFunc          types.String             `tfsdk:"trend_agg_func"`
	TrendSensitivity      types.String             `tfsdk:"trend_sensitivity"`
	TeamIDs               types.Set                `tfsdk:"team_ids"`
	ChannelIDs            types.Set                `tfsdk:"channel_ids"`
	Status                types.String             `tfsdk:"status"`
	Params                *errorMonitorParamsModel `tfsdk:"params"`
}

type errorMonitorParamsModel struct {
	Query   types.String         `tfsdk:"query"`
	Metrics []monitorMetricModel `tfsdk:"metrics"`
}

type monitorMetricModel struct {
	Name  types.String `tfsdk:"name"`
	Alias types.String `tfsdk:"alias"`
}

func (r *ErrorMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_error_monitor"
}

func (r *ErrorMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Uptrace error monitor.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Monitor ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "Project ID this monitor belongs to. Changing this forces recreation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable monitor name.",
			},
			"notify_everyone_by_email": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Notify every project member by email when the monitor fires.",
			},
			"trend_agg_func": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(trendAggFuncDefault),
				Description: "Aggregation function used to compute the trend baseline (sum, avg, median, last).",
				Validators: []validator.String{
					stringvalidator.OneOf(trendAggFuncs...),
				},
			},
			"trend_sensitivity": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(trendSensitivityDefault),
				Description: "Sensitivity level of the trend-based anomaly detector (low, medium, high).",
				Validators: []validator.String{
					stringvalidator.OneOf(trendSensitivities...),
				},
			},
			"team_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Team IDs to notify when the monitor fires. Removing this attribute clears the association server-side.",
			},
			"channel_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Notification channel IDs (uptrace_notification_channel.id). Removing this attribute clears the association server-side.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Runtime status (active, paused, firing, no_data, disabled).",
			},
			"params": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Error monitor query parameters.",
				Attributes: map[string]schema.Attribute{
					"query": schema.StringAttribute{
						Required:    true,
						Description: "MQL query expression that defines which events the monitor watches.",
					},
					"metrics": schema.ListNestedAttribute{
						Required:    true,
						Description: "Metrics referenced by the query. At least one metric must be provided.",
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Required:    true,
									Description: "Metric name.",
								},
								"alias": schema.StringAttribute{
									Optional:    true,
									Description: "Alias used in the query expression.",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *ErrorMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = client.ResourceFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *ErrorMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan errorMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := client.ParseProjectID(plan.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid project_id", err.Error())
		return
	}

	body, diags := buildErrorMonitorBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "creating error monitor", map[string]any{
		"project_id": plan.ProjectID.ValueString(),
		"name":       plan.Name.ValueString(),
	})

	out, err := r.client.API.CreateMonitor(ctx, &generated.CreateMonitorRequestOptions{
		PathParams: &generated.CreateMonitorPath{ProjectID: projectID},
		Body:       body,
	})
	if err != nil {
		resp.Diagnostics.AddError("create error monitor failed", err.Error())
		return
	}

	resp.Diagnostics.Append(monitorToErrorModel(ctx, &out.Monitor, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ErrorMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state errorMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := client.ParseProjectID(state.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid project_id", err.Error())
		return
	}
	monitorID, err := client.ParseMonitorID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid monitor id", err.Error())
		return
	}

	out, err := r.client.API.GetMonitor(ctx, &generated.GetMonitorRequestOptions{
		PathParams: &generated.GetMonitorPath{
			ProjectID: projectID,
			MonitorID: monitorID,
		},
	})
	if err != nil {
		if client.IsNotFound(err) || client.IsForbidden(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read error monitor failed", err.Error())
		return
	}

	resp.Diagnostics.Append(monitorToErrorModel(ctx, &out.Monitor, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ErrorMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan errorMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := client.ParseProjectID(plan.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid project_id", err.Error())
		return
	}
	monitorID, err := client.ParseMonitorID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid monitor id", err.Error())
		return
	}

	body, diags := buildErrorMonitorUpdateBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "updating error monitor", map[string]any{
		"id": plan.ID.ValueString(),
	})

	out, err := r.client.API.UpdateMonitor(ctx, &generated.UpdateMonitorRequestOptions{
		PathParams: &generated.UpdateMonitorPath{
			ProjectID: projectID,
			MonitorID: monitorID,
		},
		Body: body,
	})
	if err != nil {
		resp.Diagnostics.AddError("update error monitor failed", err.Error())
		return
	}

	resp.Diagnostics.Append(monitorToErrorModel(ctx, &out.Monitor, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ErrorMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state errorMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := client.ParseProjectID(state.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid project_id", err.Error())
		return
	}
	monitorID, err := client.ParseMonitorID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid monitor id", err.Error())
		return
	}

	tflog.Info(ctx, "deleting error monitor", map[string]any{
		"id": state.ID.ValueString(),
	})

	_, err = r.client.API.DeleteMonitor(ctx, &generated.DeleteMonitorRequestOptions{
		PathParams: &generated.DeleteMonitorPath{
			ProjectID: projectID,
			MonitorID: monitorID,
		},
	})
	if err != nil && !client.IsNotFound(err) && !client.IsForbidden(err) {
		resp.Diagnostics.AddError("delete error monitor failed", err.Error())
	}
}

// ImportState accepts "<project_id>:<monitor_id>".
func (r *ErrorMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client.ImportStateCompoundID(ctx, req, resp, "project_id", "monitor_id")
}

func buildErrorMonitorBody(ctx context.Context, m *errorMonitorModel) (*generated.CreateMonitorBody, diag.Diagnostics) {
	req, diags := buildErrorMonitorRequest(ctx, m)
	if diags.HasError() {
		return nil, diags
	}
	either := runtime.NewEitherFromB[generated.MetricMonitorRequest, generated.ErrorMonitorRequest](*req)
	return &generated.CreateMonitorBody{
		CreateMonitorBody_OneOf: &generated.CreateMonitorBody_OneOf{Either: either},
	}, diags
}

func buildErrorMonitorUpdateBody(ctx context.Context, m *errorMonitorModel) (*generated.UpdateMonitorBody, diag.Diagnostics) {
	req, diags := buildErrorMonitorRequest(ctx, m)
	if diags.HasError() {
		return nil, diags
	}
	either := runtime.NewEitherFromB[generated.MetricMonitorRequest, generated.ErrorMonitorRequest](*req)
	return &generated.UpdateMonitorBody{
		UpdateMonitorBody_OneOf: &generated.UpdateMonitorBody_OneOf{Either: either},
	}, diags
}

func buildErrorMonitorRequest(ctx context.Context, m *errorMonitorModel) (*generated.ErrorMonitorRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	req := &generated.ErrorMonitorRequest{
		Name: m.Name.ValueString(),
		Type: generated.ErrorMonitorRequestTypeError,
	}

	if !m.NotifyEveryoneByEmail.IsNull() && !m.NotifyEveryoneByEmail.IsUnknown() {
		v := m.NotifyEveryoneByEmail.ValueBool()
		req.NotifyEveryoneByEmail = &v
	}
	if !m.TrendAggFunc.IsNull() && !m.TrendAggFunc.IsUnknown() {
		v := generated.ErrorMonitorRequestTrendAggFunc(m.TrendAggFunc.ValueString())
		req.TrendAggFunc = &v
	}
	if !m.TrendSensitivity.IsNull() && !m.TrendSensitivity.IsUnknown() {
		v := generated.ErrorMonitorRequestTrendSensitivity(m.TrendSensitivity.ValueString())
		req.TrendSensitivity = &v
	}

	teamIDs, d := sliceFromIntSet(ctx, path.Root("team_ids"), m.TeamIDs)
	diags.Append(d...)
	if d.HasError() {
		return nil, diags
	}
	req.TeamIds = teamIDs

	channelIDs, d := sliceFromIntSet(ctx, path.Root("channel_ids"), m.ChannelIDs)
	diags.Append(d...)
	if d.HasError() {
		return nil, diags
	}
	req.ChannelIds = channelIDs

	metrics := make([]generated.MonitorMetric, len(m.Params.Metrics))
	for i, mm := range m.Params.Metrics {
		mem := generated.MonitorMetric{Name: mm.Name.ValueString()}
		if !mm.Alias.IsNull() && !mm.Alias.IsUnknown() {
			v := mm.Alias.ValueString()
			mem.Alias = &v
		}
		metrics[i] = mem
	}
	req.Params = generated.ErrorMonitorParams{
		Query:   m.Params.Query.ValueString(),
		Metrics: metrics,
	}

	return req, diags
}

func monitorToErrorModel(ctx context.Context, mon *generated.Monitor, dst *errorMonitorModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if mon.Type != generated.MonitorTypeError {
		diags.AddError(
			"monitor type mismatch",
			fmt.Sprintf(
				"uptrace_error_monitor expects a monitor of type %q, but monitor id=%d is type %q",
				generated.MonitorTypeError, mon.ID, mon.Type,
			),
		)
		return diags
	}

	dst.ID = types.StringValue(strconv.FormatInt(mon.ID, 10))
	dst.Name = types.StringValue(mon.Name)
	dst.Status = types.StringValue(string(mon.Status))

	if mon.NotifyEveryoneByEmail != nil {
		dst.NotifyEveryoneByEmail = types.BoolValue(*mon.NotifyEveryoneByEmail)
	} else {
		dst.NotifyEveryoneByEmail = types.BoolValue(false)
	}

	dst.TrendAggFunc = client.EnumToValueOrDefault(mon.TrendAggFunc, trendAggFuncDefault)
	dst.TrendSensitivity = client.EnumToValueOrDefault(mon.TrendSensitivity, trendSensitivityDefault)

	dst.TeamIDs = intSetFromSlice(mon.TeamIds, dst.TeamIDs)
	dst.ChannelIDs = intSetFromSlice(mon.ChannelIds, dst.ChannelIDs)

	params, err := decodeErrorMonitorParams(mon.Params)
	if err != nil {
		diags.AddError("decode error monitor params failed", err.Error())
		return diags
	}

	// The backend normalizes MQL (e.g. $logs → $logs{}, all.type → all.type::str)
	// before persisting. Preserving the user's input form avoids a post-apply
	// consistency error and keeps state aligned with config.
	query := types.StringValue(params.Query)
	if dst.Params != nil && !dst.Params.Query.IsNull() && !dst.Params.Query.IsUnknown() {
		query = dst.Params.Query
	}

	metrics := make([]monitorMetricModel, len(params.Metrics))
	for i, mm := range params.Metrics {
		metrics[i] = monitorMetricModel{
			Name:  types.StringValue(mm.Name),
			Alias: client.StringFromPtr(mm.Alias),
		}
	}
	dst.Params = &errorMonitorParamsModel{
		Query:   query,
		Metrics: metrics,
	}

	return diags
}

// decodeErrorMonitorParams round-trips the untyped params map from
// generated.Monitor into a strongly-typed ErrorMonitorParams. Safe for error
// monitor (2 fields) but drops unknown keys silently — a real limitation for
// richer shapes like metric monitor. Upstream fix is making the response
// Monitor type carry a discriminated oneOf, same as CreateMonitorBody.
func decodeErrorMonitorParams(raw map[string]any) (*generated.ErrorMonitorParams, error) {
	buf, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal params map: %w", err)
	}
	var out generated.ErrorMonitorParams
	if err := json.Unmarshal(buf, &out); err != nil {
		return nil, fmt.Errorf("unmarshal params: %w", err)
	}
	return &out, nil
}

// intSetFromSlice maps a []int API response to a types.Set of stringified
// IDs. An explicit empty prior set round-trips as empty; otherwise empty
// API → SetNull.
//
// IDs are strings on the TF side so set values can reference string-typed
// resource IDs directly (e.g. uptrace_notification_channel.x.id) without a
// tonumber() wrapper. Sets are used (not lists) because recipient IDs have
// no meaningful order and a list would churn on every plan.
func intSetFromSlice(xs []int, prior types.Set) types.Set {
	if len(xs) == 0 {
		if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) == 0 {
			return prior
		}
		return types.SetNull(types.StringType)
	}
	vals := make([]attr.Value, len(xs))
	for i, x := range xs {
		vals[i] = types.StringValue(strconv.Itoa(x))
	}
	s, _ := types.SetValue(types.StringType, vals)
	return s
}

// sliceFromIntSet decodes a types.Set of string-encoded IDs into []int.
// A non-decimal element emits an attribute diagnostic.
func sliceFromIntSet(ctx context.Context, attrPath path.Path, s types.Set) ([]int, diag.Diagnostics) {
	var diags diag.Diagnostics
	if s.IsNull() || s.IsUnknown() {
		return nil, diags
	}
	var vals []types.String
	diags.Append(s.ElementsAs(ctx, &vals, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]int, 0, len(vals))
	for _, v := range vals {
		str := v.ValueString()
		n, err := strconv.Atoi(str)
		if err != nil {
			diags.AddAttributeError(
				attrPath,
				"invalid ID",
				fmt.Sprintf("expected a decimal integer, got %q: %s", str, err.Error()),
			)
			return nil, diags
		}
		out = append(out, n)
	}
	return out, diags
}
