package monitor

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

func TestBuildErrorMonitorRequest_minimal(t *testing.T) {
	m := &errorMonitorModel{
		Name:                  types.StringValue("err-monitor"),
		NotifyEveryoneByEmail: types.BoolValue(false),
		TrendAggFunc:          types.StringValue("sum"),
		TrendSensitivity:      types.StringValue("medium"),
		TeamIDs:               types.SetNull(types.StringType),
		ChannelIDs:            types.SetNull(types.StringType),
		Params: &errorParamsModel{
			Metrics: []monitorMetricModel{
				{Name: types.StringValue("uptrace_tracing_logs"), Alias: types.StringValue("$logs")},
			},
			Query: types.StringValue("sum($logs) | where true"),
		},
	}

	req, diags := buildErrorMonitorRequest(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, "err-monitor", req.Name)
	require.Equal(t, generated.ErrorMonitorRequestTypeError, req.Type)
	require.NotNil(t, req.NotifyEveryoneByEmail)
	require.False(t, *req.NotifyEveryoneByEmail)
	require.Nil(t, req.TeamIds)
	require.Nil(t, req.ChannelIds)
	require.Equal(t, "sum($logs) | where true", req.Params.Query)
	require.Len(t, req.Params.Metrics, 1)
	require.Equal(t, "uptrace_tracing_logs", req.Params.Metrics[0].Name)
	require.Equal(t, "$logs", *req.Params.Metrics[0].Alias)
}

func TestBuildErrorMonitorRequest_withIDs(t *testing.T) {
	teamIDs, _ := types.SetValueFrom(context.Background(), types.StringType, []string{"5", "3"})
	channelIDs, _ := types.SetValueFrom(context.Background(), types.StringType, []string{"11"})
	m := &errorMonitorModel{
		Name:       types.StringValue("with-ids"),
		TeamIDs:    teamIDs,
		ChannelIDs: channelIDs,
		Params: &errorParamsModel{
			Metrics: []monitorMetricModel{
				{Name: types.StringValue("metric_x"), Alias: types.StringNull()},
			},
			Query: types.StringValue("sum($x)"),
		},
	}
	req, diags := buildErrorMonitorRequest(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.ElementsMatch(t, []int{5, 3}, req.TeamIds)
	require.Equal(t, []int{11}, req.ChannelIds)
	require.Nil(t, req.Params.Metrics[0].Alias)
}

func TestBuildErrorCreateBody_wrapsInEitherB(t *testing.T) {
	m := &errorMonitorModel{
		Name: types.StringValue("wrap"),
		Params: &errorParamsModel{
			Metrics: []monitorMetricModel{
				{Name: types.StringValue("metric_x"), Alias: types.StringValue("$x")},
			},
			Query: types.StringValue("sum($x)"),
		},
	}
	body, diags := buildErrorCreateBody(context.Background(), m)
	require.False(t, diags.HasError())
	require.NotNil(t, body.CreateMonitorBody_OneOf)
	require.True(t, body.CreateMonitorBody_OneOf.IsB(), "error monitor should be Either's B variant")
}

func TestApplyErrorMonitor_preservesPriorQuery(t *testing.T) {
	mon := &generated.Monitor{
		ID:     77,
		Name:   "err",
		Type:   generated.MonitorTypeError,
		Status: generated.Active,
		Params: map[string]any{
			"query": "sum($logs{}) | where _system::str = \"log:error\"",
			"metrics": []any{
				map[string]any{"name": "uptrace_tracing_logs", "alias": "$logs"},
			},
		},
	}
	dst := &errorMonitorModel{
		Params: &errorParamsModel{
			Query: types.StringValue(`sum($logs) | where _system = "log:error"`),
		},
	}
	diags := applyErrorMonitorToModel(mon, dst)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, `sum($logs) | where _system = "log:error"`, dst.Params.Query.ValueString())
	require.Len(t, dst.Params.Metrics, 1)
}

func TestApplyErrorMonitor_takesAPIQueryWhenPriorNull(t *testing.T) {
	mon := &generated.Monitor{
		ID:     77,
		Name:   "err",
		Type:   generated.MonitorTypeError,
		Status: generated.Active,
		Params: map[string]any{
			"query":   "sum($logs)",
			"metrics": []any{map[string]any{"name": "uptrace_tracing_logs", "alias": "$logs"}},
		},
	}
	dst := &errorMonitorModel{}

	diags := applyErrorMonitorToModel(mon, dst)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, "sum($logs)", dst.Params.Query.ValueString())
}

func TestApplyErrorMonitor_idSetsMapUnordered(t *testing.T) {
	mon := &generated.Monitor{
		ID:         77,
		Name:       "err",
		Type:       generated.MonitorTypeError,
		Status:     generated.Active,
		TeamIds:    []int{9, 1, 5},
		ChannelIds: []int{3, 2},
		Params: map[string]any{
			"query":   "sum($logs)",
			"metrics": []any{map[string]any{"name": "uptrace_tracing_logs", "alias": "$logs"}},
		},
	}
	dst := &errorMonitorModel{}

	diags := applyErrorMonitorToModel(mon, dst)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.False(t, dst.TeamIDs.IsNull())
	var team []string
	dst.TeamIDs.ElementsAs(context.Background(), &team, false)
	require.ElementsMatch(t, []string{"1", "5", "9"}, team)

	var ch []string
	dst.ChannelIDs.ElementsAs(context.Background(), &ch, false)
	require.ElementsMatch(t, []string{"2", "3"}, ch)
}

func TestApplyErrorMonitor_rejectsTypeMismatch(t *testing.T) {
	mon := &generated.Monitor{
		ID:     77,
		Name:   "not-an-error",
		Type:   generated.MonitorTypeMetric,
		Status: generated.Active,
	}
	dst := &errorMonitorModel{}

	diags := applyErrorMonitorToModel(mon, dst)
	require.True(t, diags.HasError(), "expected error when API type does not match resource type")
}
