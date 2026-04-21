package monitor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/client"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

type errorParamsModel struct {
	Query   types.String         `tfsdk:"query"`
	Metrics []monitorMetricModel `tfsdk:"metrics"`
}

func buildErrorMonitorRequest(ctx context.Context, m *monitorModel) (*generated.ErrorMonitorRequest, diag.Diagnostics) {
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

	teamIDs, d := client.SliceFromIntSet(ctx, path.Root("team_ids"), m.TeamIDs)
	diags.Append(d...)
	if d.HasError() {
		return nil, diags
	}
	req.TeamIds = teamIDs

	channelIDs, d := client.SliceFromIntSet(ctx, path.Root("channel_ids"), m.ChannelIDs)
	diags.Append(d...)
	if d.HasError() {
		return nil, diags
	}
	req.ChannelIds = channelIDs

	if m.ParamsError == nil {
		diags.AddAttributeError(
			path.Root("params_error"),
			"params_error is required",
			`params_error block must be set when type is "error".`,
		)
		return nil, diags
	}
	metrics := make([]generated.MonitorMetric, len(m.ParamsError.Metrics))
	for i, mm := range m.ParamsError.Metrics {
		mem := generated.MonitorMetric{Name: mm.Name.ValueString()}
		if !mm.Alias.IsNull() && !mm.Alias.IsUnknown() {
			v := mm.Alias.ValueString()
			mem.Alias = &v
		}
		metrics[i] = mem
	}
	req.Params = generated.ErrorMonitorParams{
		Query:   m.ParamsError.Query.ValueString(),
		Metrics: metrics,
	}

	return req, diags
}

func applyErrorMonitorToModel(mon *generated.Monitor, dst *monitorModel) diag.Diagnostics {
	var diags diag.Diagnostics

	params, err := decodeErrorMonitorParams(mon.Params)
	if err != nil {
		diags.AddError("decode error monitor params failed", err.Error())
		return diags
	}

	var priorQuery types.String
	if dst.ParamsError != nil {
		priorQuery = dst.ParamsError.Query
	}

	metrics := make([]monitorMetricModel, len(params.Metrics))
	for i, mm := range params.Metrics {
		metrics[i] = monitorMetricModel{
			Name:  types.StringValue(mm.Name),
			Alias: client.StringFromPtr(mm.Alias),
		}
	}
	dst.ParamsError = &errorParamsModel{
		Query:   client.PreferPrior(priorQuery, types.StringValue(params.Query)),
		Metrics: metrics,
	}
	dst.ParamsMetric = nil

	return diags
}

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
