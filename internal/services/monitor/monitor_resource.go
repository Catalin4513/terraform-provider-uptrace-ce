package monitor

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                   = &MonitorResource{}
	_ resource.ResourceWithConfigure      = &MonitorResource{}
	_ resource.ResourceWithImportState    = &MonitorResource{}
	_ resource.ResourceWithValidateConfig = &MonitorResource{}
)

var (
	monitorTypes       = []string{"error", "metric"}
	trendAggFuncs      = []string{"sum", "avg", "median", "last"}
	trendSensitivities = []string{"low", "medium", "high"}
	absentPointsKinds  = []string{"ignore", "alert", "zero"}
	tolerances         = []string{"low", "medium", "high"}
)

const (
	trendAggFuncDefault     = "sum"
	trendSensitivityDefault = "medium"
)

type MonitorResource struct {
	client *client.Client
}

func NewMonitorResource() resource.Resource {
	return &MonitorResource{}
}

type monitorModel struct {
	ID                    types.String `tfsdk:"id"`
	ProjectID             types.String `tfsdk:"project_id"`
	Name                  types.String `tfsdk:"name"`
	Type                  types.String `tfsdk:"type"`
	NotifyEveryoneByEmail types.Bool   `tfsdk:"notify_everyone_by_email"`
	TrendAggFunc          types.String `tfsdk:"trend_agg_func"`
	TrendSensitivity      types.String `tfsdk:"trend_sensitivity"`
	TeamIDs               types.Set    `tfsdk:"team_ids"`
	ChannelIDs            types.Set    `tfsdk:"channel_ids"`
	Status                types.String `tfsdk:"status"`

	ParamsError  *errorParamsModel  `tfsdk:"params_error"`
	ParamsMetric *metricParamsModel `tfsdk:"params_metric"`
}

// monitorMetricModel maps a single entry of the metrics list shared by both
// error and metric monitor params blocks.
type monitorMetricModel struct {
	Name  types.String `tfsdk:"name"`
	Alias types.String `tfsdk:"alias"`
}

func (r *MonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (r *MonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Uptrace monitor. Set `type` to `error` or `metric` and configure the matching `params_error` / `params_metric` block.",
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
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Monitor type: `error` or `metric`. Changing this forces recreation.",
				Validators: []validator.String{
					stringvalidator.OneOf(monitorTypes...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
			"params_error":  errorParamsAttribute(),
			"params_metric": metricParamsAttribute(),
		},
	}
}

func errorParamsAttribute() schema.Attribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Error monitor query parameters. Required when `type = \"error\"`.",
		Attributes: map[string]schema.Attribute{
			"query": schema.StringAttribute{
				Required:    true,
				Description: "MQL query expression that defines which events the monitor watches.",
			},
			"metrics": monitorMetricsAttribute(),
		},
	}
}

func metricParamsAttribute() schema.Attribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Metric monitor query parameters. Required when `type = \"metric\"`.",
		Attributes: map[string]schema.Attribute{
			"query": schema.StringAttribute{
				Required:    true,
				Description: "MQL query expression.",
			},
			"metrics": monitorMetricsAttribute(),
			"column": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Query result column that the detector evaluates. When unset, the server derives one from the query and the provider preserves the null state.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Optional:    true,
						Description: "Name of the query result column being monitored.",
					},
					"unit": schema.StringAttribute{
						Optional:    true,
						Description: "Display unit for the column values (e.g. \"milliseconds\", \"bytes\").",
					},
				},
			},
			"resolution": schema.Float64Attribute{
				Optional:    true,
				Description: "Evaluation resolution in milliseconds.",
			},
			"num_eval_points": schema.Int64Attribute{
				Optional:    true,
				Description: "Number of consecutive evaluation points that must breach the threshold.",
			},
			"absent_points": schema.StringAttribute{
				Optional:    true,
				Description: "How to treat gaps in the query output (ignore, alert, zero).",
				Validators: []validator.String{
					stringvalidator.OneOf(absentPointsKinds...),
				},
			},
			"time_offset": schema.Float64Attribute{
				Optional:    true,
				Description: "Time offset in milliseconds applied to the query before evaluation.",
			},
			"detector": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Detector configuration. Exactly one of `manual` or `auto` must be set.",
				Attributes: map[string]schema.Attribute{
					"manual": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "Manual threshold detector.",
						Attributes: map[string]schema.Attribute{
							"min_value": schema.Float64Attribute{Optional: true, Description: "Alert when value falls below this threshold."},
							"max_value": schema.Float64Attribute{Optional: true, Description: "Alert when value rises above this threshold."},
							"recovery": schema.SingleNestedAttribute{
								Optional:    true,
								Description: "Hysteresis thresholds used to clear an active alert.",
								Attributes: map[string]schema.Attribute{
									"min_value": schema.Float64Attribute{Optional: true, Description: "Clear the alert once the value rises back above this threshold."},
									"max_value": schema.Float64Attribute{Optional: true, Description: "Clear the alert once the value falls back below this threshold."},
								},
							},
						},
					},
					"auto": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "Automatic trend-based detector.",
						Attributes: map[string]schema.Attribute{
							"tolerance": schema.StringAttribute{
								Optional:    true,
								Description: "Deviation tolerance (low, medium, high).",
								Validators: []validator.String{
									stringvalidator.OneOf(tolerances...),
								},
							},
							"training_period":  schema.Float64Attribute{Optional: true, Description: "Training period in milliseconds."},
							"min_dev_fraction": schema.Float64Attribute{Optional: true, Description: "Minimum deviation as a fraction of the baseline."},
							"min_dev_absolute": schema.Float64Attribute{Optional: true, Description: "Minimum absolute deviation from the baseline."},
						},
					},
				},
			},
		},
	}
}

func monitorMetricsAttribute() schema.Attribute {
	return schema.ListNestedAttribute{
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
	}
}

func (r *MonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = client.ResourceFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *MonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := client.ParseProjectID(plan.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid project_id", err.Error())
		return
	}

	body, diags := buildCreateBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "creating monitor", map[string]any{
		"project_id": plan.ProjectID.ValueString(),
		"name":       plan.Name.ValueString(),
		"type":       plan.Type.ValueString(),
	})

	out, err := r.client.API.CreateMonitor(ctx, &generated.CreateMonitorRequestOptions{
		PathParams: &generated.CreateMonitorPath{ProjectID: projectID},
		Body:       body,
	})
	if err != nil {
		resp.Diagnostics.AddError("create monitor failed", err.Error())
		return
	}

	resp.Diagnostics.Append(applyMonitorToModel(&out.Monitor, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitorModel
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
		resp.Diagnostics.AddError("read monitor failed", err.Error())
		return
	}

	resp.Diagnostics.Append(applyMonitorToModel(&out.Monitor, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan monitorModel
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

	body, diags := buildUpdateBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "updating monitor", map[string]any{
		"id":   plan.ID.ValueString(),
		"type": plan.Type.ValueString(),
	})

	out, err := r.client.API.UpdateMonitor(ctx, &generated.UpdateMonitorRequestOptions{
		PathParams: &generated.UpdateMonitorPath{
			ProjectID: projectID,
			MonitorID: monitorID,
		},
		Body: body,
	})
	if err != nil {
		resp.Diagnostics.AddError("update monitor failed", err.Error())
		return
	}

	resp.Diagnostics.Append(applyMonitorToModel(&out.Monitor, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitorModel
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

	tflog.Info(ctx, "deleting monitor", map[string]any{
		"id": state.ID.ValueString(),
	})

	_, err = r.client.API.DeleteMonitor(ctx, &generated.DeleteMonitorRequestOptions{
		PathParams: &generated.DeleteMonitorPath{
			ProjectID: projectID,
			MonitorID: monitorID,
		},
	})
	if err != nil && !client.IsNotFound(err) && !client.IsForbidden(err) {
		resp.Diagnostics.AddError("delete monitor failed", err.Error())
	}
}

// ImportState accepts "<project_id>:<monitor_id>".
func (r *MonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client.ImportStateCompoundID(ctx, req, resp, "project_id", "monitor_id")
}

// --- build / apply dispatchers ---

func buildCreateBody(ctx context.Context, m *monitorModel) (*generated.CreateMonitorBody, diag.Diagnostics) {
	either, diags := buildMonitorEither(ctx, m)
	if diags.HasError() {
		return nil, diags
	}
	return &generated.CreateMonitorBody{
		CreateMonitorBody_OneOf: &generated.CreateMonitorBody_OneOf{Either: either},
	}, diags
}

func buildUpdateBody(ctx context.Context, m *monitorModel) (*generated.UpdateMonitorBody, diag.Diagnostics) {
	either, diags := buildMonitorEither(ctx, m)
	if diags.HasError() {
		return nil, diags
	}
	return &generated.UpdateMonitorBody{
		UpdateMonitorBody_OneOf: &generated.UpdateMonitorBody_OneOf{Either: either},
	}, diags
}

func buildMonitorEither(ctx context.Context, m *monitorModel) (runtime.Either[generated.MetricMonitorRequest, generated.ErrorMonitorRequest], diag.Diagnostics) {
	var diags diag.Diagnostics
	var zero runtime.Either[generated.MetricMonitorRequest, generated.ErrorMonitorRequest]

	switch m.Type.ValueString() {
	case "error":
		req, d := buildErrorMonitorRequest(ctx, m)
		diags.Append(d...)
		if d.HasError() {
			return zero, diags
		}
		return runtime.NewEitherFromB[generated.MetricMonitorRequest, generated.ErrorMonitorRequest](*req), diags
	case "metric":
		req, d := buildMetricMonitorRequest(ctx, m)
		diags.Append(d...)
		if d.HasError() {
			return zero, diags
		}
		return runtime.NewEitherFromA[generated.MetricMonitorRequest, generated.ErrorMonitorRequest](*req), diags
	default:
		diags.AddError("invalid monitor type", fmt.Sprintf("expected error or metric, got %q", m.Type.ValueString()))
		return zero, diags
	}
}

// applyMonitorToModel writes shared fields from the API response into the
// model, then dispatches type-specific params mapping.
func applyMonitorToModel(mon *generated.Monitor, dst *monitorModel) diag.Diagnostics {
	var diags diag.Diagnostics

	expected := dst.Type.ValueString()
	actual := string(mon.Type)
	if expected != "" && expected != actual {
		diags.AddError(
			"monitor type mismatch",
			fmt.Sprintf("expected monitor type %q, API returned %q for monitor id=%d", expected, actual, mon.ID),
		)
		return diags
	}

	dst.ID = types.StringValue(strconv.FormatInt(mon.ID, 10))
	dst.Name = types.StringValue(mon.Name)
	dst.Type = types.StringValue(actual)
	dst.Status = types.StringValue(string(mon.Status))

	if mon.NotifyEveryoneByEmail != nil {
		dst.NotifyEveryoneByEmail = types.BoolValue(*mon.NotifyEveryoneByEmail)
	} else {
		dst.NotifyEveryoneByEmail = types.BoolValue(false)
	}

	dst.TrendAggFunc = client.EnumToValueOrDefault(mon.TrendAggFunc, trendAggFuncDefault)
	dst.TrendSensitivity = client.EnumToValueOrDefault(mon.TrendSensitivity, trendSensitivityDefault)

	dst.TeamIDs = client.IntSetFromSlice(mon.TeamIds, dst.TeamIDs)
	dst.ChannelIDs = client.IntSetFromSlice(mon.ChannelIds, dst.ChannelIDs)

	switch mon.Type {
	case generated.MonitorTypeError:
		diags.Append(applyErrorMonitorToModel(mon, dst)...)
	case generated.MonitorTypeMetric:
		diags.Append(applyMetricMonitorToModel(mon, dst)...)
	default:
		diags.AddError("unknown monitor type", fmt.Sprintf("API returned unsupported monitor type %q", mon.Type))
	}
	return diags
}
