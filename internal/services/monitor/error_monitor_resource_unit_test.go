package monitor

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/client"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

func TestBuildErrorMonitorRequest_minimal(t *testing.T) {
	m := &errorMonitorModel{
		Name:                  types.StringValue("err-monitor"),
		NotifyEveryoneByEmail: types.BoolValue(false),
		TrendAggFunc:          types.StringValue("sum"),
		TrendSensitivity:      types.StringValue("medium"),
		TeamIDs:               types.ListNull(types.StringType),
		ChannelIDs:            types.ListNull(types.StringType),
		Params: &errorMonitorParamsModel{
			Query: types.StringValue("sum($logs) | where true"),
			Metrics: []monitorMetricModel{
				{Name: types.StringValue("uptrace_tracing_logs"), Alias: types.StringValue("$logs")},
			},
		},
	}

	req, diags := buildErrorMonitorRequest(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, "err-monitor", req.Name)
	require.Equal(t, generated.ErrorMonitorRequestTypeError, req.Type)
	require.NotNil(t, req.NotifyEveryoneByEmail)
	require.False(t, *req.NotifyEveryoneByEmail)
	require.NotNil(t, req.TrendAggFunc)
	require.Equal(t, "sum", string(*req.TrendAggFunc))
	require.Nil(t, req.TeamIds)
	require.Nil(t, req.ChannelIds)
	require.Equal(t, "sum($logs) | where true", req.Params.Query)
	require.Len(t, req.Params.Metrics, 1)
	require.Equal(t, "uptrace_tracing_logs", req.Params.Metrics[0].Name)
	require.NotNil(t, req.Params.Metrics[0].Alias)
	require.Equal(t, "$logs", *req.Params.Metrics[0].Alias)
}

func TestBuildErrorMonitorRequest_withIDs(t *testing.T) {
	teamIDs, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"5", "3"})
	channelIDs, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"11"})
	m := &errorMonitorModel{
		Name:       types.StringValue("with-ids"),
		TeamIDs:    teamIDs,
		ChannelIDs: channelIDs,
		Params: &errorMonitorParamsModel{
			Query: types.StringValue("sum($x)"),
			Metrics: []monitorMetricModel{
				{Name: types.StringValue("metric_x"), Alias: types.StringNull()},
			},
		},
	}

	req, diags := buildErrorMonitorRequest(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, []int{5, 3}, req.TeamIds)
	require.Equal(t, []int{11}, req.ChannelIds)
	require.Nil(t, req.Params.Metrics[0].Alias)
}

func TestBuildErrorMonitorBody_wrapsInEitherB(t *testing.T) {
	m := &errorMonitorModel{
		Name: types.StringValue("wrap"),
		Params: &errorMonitorParamsModel{
			Query: types.StringValue("sum($x)"),
			Metrics: []monitorMetricModel{
				{Name: types.StringValue("metric_x"), Alias: types.StringValue("$x")},
			},
		},
	}
	body, diags := buildErrorMonitorBody(context.Background(), m)
	require.False(t, diags.HasError())
	require.NotNil(t, body.CreateMonitorBody_OneOf)
	require.True(t, body.CreateMonitorBody_OneOf.IsB(), "error monitor should be Either's B variant")
}

func TestMonitorToErrorModel_preservesPriorQuery(t *testing.T) {
	mon := &generated.Monitor{
		ID:     77,
		Name:   "err",
		Type:   generated.MonitorTypeError,
		Status: generated.Active,
		Params: map[string]any{
			// Server's normalized form — mapper must NOT use this.
			"query": "sum($logs{}) | where _system::str = \"log:error\"",
			"metrics": []any{
				map[string]any{"name": "uptrace_tracing_logs", "alias": "$logs"},
			},
		},
	}
	dst := &errorMonitorModel{
		Params: &errorMonitorParamsModel{
			Query: types.StringValue("sum($logs) | where _system = \"log:error\""),
		},
	}
	diags := monitorToErrorModel(context.Background(), mon, dst)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, `sum($logs) | where _system = "log:error"`, dst.Params.Query.ValueString())
	require.Len(t, dst.Params.Metrics, 1)
}

func TestMonitorToErrorModel_takesAPIQueryWhenPriorNull(t *testing.T) {
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
	dst := &errorMonitorModel{} // no prior state

	diags := monitorToErrorModel(context.Background(), mon, dst)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, "sum($logs)", dst.Params.Query.ValueString())
}

func TestMonitorToErrorModel_idListsSorted(t *testing.T) {
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

	diags := monitorToErrorModel(context.Background(), mon, dst)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.False(t, dst.TeamIDs.IsNull())
	var team []string
	dst.TeamIDs.ElementsAs(context.Background(), &team, false)
	require.Equal(t, []string{"1", "5", "9"}, team)

	var ch []string
	dst.ChannelIDs.ElementsAs(context.Background(), &ch, false)
	require.Equal(t, []string{"2", "3"}, ch)
}

func TestMonitorToErrorModel_emptyIDListsCollapseToNull(t *testing.T) {
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
	dst := &errorMonitorModel{
		TeamIDs:    types.ListNull(types.StringType),
		ChannelIDs: types.ListNull(types.StringType),
	}

	diags := monitorToErrorModel(context.Background(), mon, dst)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.True(t, dst.TeamIDs.IsNull())
	require.True(t, dst.ChannelIDs.IsNull())
}

func TestIntListFromSlice_preservesExplicitEmpty(t *testing.T) {
	priorEmpty, _ := types.ListValueFrom(context.Background(), types.StringType, []string{})
	got := intListFromSlice(nil, priorEmpty)
	require.False(t, got.IsNull(), "explicit prior empty list must round-trip")
	require.Equal(t, 0, len(got.Elements()))
}

func TestSliceFromIntList_nullReturnsNil(t *testing.T) {
	out, diags := sliceFromIntList(context.Background(), path.Root("team_ids"), types.ListNull(types.StringType))
	require.False(t, diags.HasError())
	require.Nil(t, out)
}

func TestSliceFromIntList_invalidElement(t *testing.T) {
	bad, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"12", "not-a-number"})
	_, diags := sliceFromIntList(context.Background(), path.Root("team_ids"), bad)
	require.True(t, diags.HasError())
}

func TestParseMonitorID(t *testing.T) {
	id, err := client.ParseMonitorID("123")
	require.NoError(t, err)
	require.Equal(t, int64(123), id)

	_, err = client.ParseMonitorID("abc")
	require.Error(t, err)
}

// verify the Either helper's B-variant matches our expected type.
func TestEitherB_errorMonitorRequest(t *testing.T) {
	req := generated.ErrorMonitorRequest{Name: "x", Type: generated.ErrorMonitorRequestTypeError}
	e := runtime.NewEitherFromB[generated.MetricMonitorRequest, generated.ErrorMonitorRequest](req)
	require.True(t, e.IsB())
	require.Equal(t, "x", e.B.Name)
}
