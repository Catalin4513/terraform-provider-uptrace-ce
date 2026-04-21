package monitor

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

// --- error monitor: builder ---

func TestBuildErrorMonitorRequest_minimal(t *testing.T) {
	m := &monitorModel{
		Name:                  types.StringValue("err-monitor"),
		Type:                  types.StringValue("error"),
		NotifyEveryoneByEmail: types.BoolValue(false),
		TrendAggFunc:          types.StringValue("sum"),
		TrendSensitivity:      types.StringValue("medium"),
		TeamIDs:               types.SetNull(types.StringType),
		ChannelIDs:            types.SetNull(types.StringType),
		ParamsError: &errorParamsModel{
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
	m := &monitorModel{
		Name:       types.StringValue("with-ids"),
		Type:       types.StringValue("error"),
		TeamIDs:    teamIDs,
		ChannelIDs: channelIDs,
		ParamsError: &errorParamsModel{
			Query: types.StringValue("sum($x)"),
			Metrics: []monitorMetricModel{
				{Name: types.StringValue("metric_x"), Alias: types.StringNull()},
			},
		},
	}
	req, diags := buildErrorMonitorRequest(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.ElementsMatch(t, []int{5, 3}, req.TeamIds)
	require.Equal(t, []int{11}, req.ChannelIds)
	require.Nil(t, req.Params.Metrics[0].Alias)
}

// The Either ordering in CreateMonitorBody_OneOf is
// Either[MetricMonitorRequest, ErrorMonitorRequest] — metric is A, error is
// B. A swap during codegen would silently route one type's payload as the
// other; this test guards against that.
func TestBuildCreateBody_variantByType(t *testing.T) {
	tests := []struct {
		name  string
		model *monitorModel
		isA   bool
	}{
		{
			name: "metric is A",
			model: &monitorModel{
				Name: types.StringValue("m"),
				Type: types.StringValue("metric"),
				ParamsMetric: &metricParamsModel{
					Query: types.StringValue("sum($x)"),
					Metrics: []monitorMetricModel{
						{Name: types.StringValue("m"), Alias: types.StringValue("$x")},
					},
					Detector: &detectorModel{Auto: &autoDetectorModel{}},
				},
			},
			isA: true,
		},
		{
			name: "error is B",
			model: &monitorModel{
				Name: types.StringValue("wrap"),
				Type: types.StringValue("error"),
				ParamsError: &errorParamsModel{
					Query: types.StringValue("sum($x)"),
					Metrics: []monitorMetricModel{
						{Name: types.StringValue("metric_x"), Alias: types.StringValue("$x")},
					},
				},
			},
			isA: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, diags := buildCreateBody(context.Background(), tc.model)
			require.False(t, diags.HasError())
			require.NotNil(t, body.CreateMonitorBody_OneOf)
			require.Equal(t, tc.isA, body.CreateMonitorBody_OneOf.IsA())
			require.Equal(t, !tc.isA, body.CreateMonitorBody_OneOf.IsB())
		})
	}
}

// --- error monitor: mapper ---

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
	dst := &monitorModel{
		Type: types.StringValue("error"),
		ParamsError: &errorParamsModel{
			Query: types.StringValue("sum($logs) | where _system = \"log:error\""),
		},
	}
	diags := applyMonitorToModel(mon, dst)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, `sum($logs) | where _system = "log:error"`, dst.ParamsError.Query.ValueString())
	require.Len(t, dst.ParamsError.Metrics, 1)
	require.Nil(t, dst.ParamsMetric)
}

func TestApplyMonitorToModel_rejectsTypeMismatch(t *testing.T) {
	mon := &generated.Monitor{
		ID:     77,
		Name:   "not-an-error",
		Type:   generated.MonitorTypeMetric,
		Status: generated.Active,
	}
	dst := &monitorModel{Type: types.StringValue("error")}

	diags := applyMonitorToModel(mon, dst)
	require.True(t, diags.HasError(), "expected error when API type does not match planned type")
}

// --- metric monitor: builder ---

func TestBuildMetricMonitorRequest_autoDetector(t *testing.T) {
	m := &monitorModel{
		Name:             types.StringValue("latency"),
		Type:             types.StringValue("metric"),
		TrendAggFunc:     types.StringValue("avg"),
		TrendSensitivity: types.StringValue("high"),
		TeamIDs:          types.SetNull(types.StringType),
		ChannelIDs:       types.SetNull(types.StringType),
		ParamsMetric: &metricParamsModel{
			Query: types.StringValue("avg($http_server)"),
			Metrics: []monitorMetricModel{
				{Name: types.StringValue("http_server_duration"), Alias: types.StringValue("$http_server")},
			},
			Column:       &columnModel{Name: types.StringValue("value"), Unit: types.StringValue("milliseconds")},
			Resolution:   types.Float64Value(60000),
			AbsentPoints: types.StringValue("ignore"),
			Detector: &detectorModel{
				Auto: &autoDetectorModel{
					Tolerance:      types.StringValue("medium"),
					TrainingPeriod: types.Float64Value(86400000),
				},
			},
		},
	}

	req, diags := buildMetricMonitorRequest(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, "latency", req.Name)
	require.Equal(t, generated.MetricMonitorRequestTypeMetric, req.Type)
	require.NotNil(t, req.Params.Column)
	require.Equal(t, "value", *req.Params.Column.Name)
	require.NotNil(t, req.Params.Resolution)
	require.InDelta(t, float32(60000), *req.Params.Resolution, 0.01)
	require.NotNil(t, req.Params.AbsentPoints)
	require.Equal(t, generated.Ignore, *req.Params.AbsentPoints)
	require.Equal(t, generated.Auto, req.Params.Detector.Type)
	require.True(t, req.Params.Detector.Params.DetectorConfig_Params_OneOf.IsB())
	require.NotNil(t, req.Params.Detector.Params.DetectorConfig_Params_OneOf.B.Tolerance)
	require.Equal(t, generated.AutoDetectorParamsToleranceMedium, *req.Params.Detector.Params.DetectorConfig_Params_OneOf.B.Tolerance)
}

func TestBuildMetricMonitorRequest_manualDetector(t *testing.T) {
	m := &monitorModel{
		Name:       types.StringValue("thresh"),
		Type:       types.StringValue("metric"),
		TeamIDs:    types.SetNull(types.StringType),
		ChannelIDs: types.SetNull(types.StringType),
		ParamsMetric: &metricParamsModel{
			Query: types.StringValue("sum($x)"),
			Metrics: []monitorMetricModel{
				{Name: types.StringValue("m"), Alias: types.StringValue("$x")},
			},
			Detector: &detectorModel{
				Manual: &manualDetectorModel{
					MinValue: types.Float64Value(1),
					MaxValue: types.Float64Value(100),
					Recovery: &recoveryModel{
						MinValue: types.Float64Value(10),
						MaxValue: types.Float64Value(90),
					},
				},
			},
		},
	}

	req, diags := buildMetricMonitorRequest(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, generated.Manual, req.Params.Detector.Type)
	require.True(t, req.Params.Detector.Params.DetectorConfig_Params_OneOf.IsA())
	manual := req.Params.Detector.Params.DetectorConfig_Params_OneOf.A
	require.InDelta(t, float32(1), *manual.MinValue, 0.01)
	require.InDelta(t, float32(100), *manual.MaxValue, 0.01)
	require.NotNil(t, manual.Recovery)
	require.InDelta(t, float32(10), *manual.Recovery.MinValue, 0.01)
}

// --- metric monitor: mapper ---

func TestApplyMetricMonitor_autoDetector(t *testing.T) {
	mon := &generated.Monitor{
		ID:     101,
		Name:   "latency",
		Type:   generated.MonitorTypeMetric,
		Status: generated.Active,
		Params: map[string]any{
			"query":   "avg($http)",
			"metrics": []any{map[string]any{"name": "http", "alias": "$http"}},
			"column":  map[string]any{"name": "value", "unit": "milliseconds"},
			"detector": map[string]any{
				"type": "auto",
				"params": map[string]any{
					"tolerance":      "high",
					"trainingPeriod": 86400000,
				},
			},
		},
	}
	// Prior state with column explicitly set; preservation logic takes the
	// API value through.
	dst := &monitorModel{
		Type: types.StringValue("metric"),
		ParamsMetric: &metricParamsModel{
			Column: &columnModel{Name: types.StringValue("value"), Unit: types.StringValue("milliseconds")},
		},
	}

	diags := applyMonitorToModel(mon, dst)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.NotNil(t, dst.ParamsMetric)
	require.Nil(t, dst.ParamsError)
	require.NotNil(t, dst.ParamsMetric.Column)
	require.Equal(t, "value", dst.ParamsMetric.Column.Name.ValueString())
	require.NotNil(t, dst.ParamsMetric.Detector.Auto)
	require.Nil(t, dst.ParamsMetric.Detector.Manual)
	require.Equal(t, "high", dst.ParamsMetric.Detector.Auto.Tolerance.ValueString())
}

// When prior state does not set column (typical import / user didn't
// specify one), the server-derived column is preserved as null to avoid
// a post-apply consistency error.
func TestApplyMetricMonitor_columnPreservedAsNull(t *testing.T) {
	mon := &generated.Monitor{
		ID:     101,
		Name:   "latency",
		Type:   generated.MonitorTypeMetric,
		Status: generated.Active,
		Params: map[string]any{
			"query":         "avg($http)",
			"metrics":       []any{map[string]any{"name": "http", "alias": "$http"}},
			"column":        map[string]any{"name": "derived-by-server", "unit": "milliseconds"},
			"resolution":    60000,
			"absentPoints":  "alert",
			"numEvalPoints": 5,
			"detector": map[string]any{
				"type":   "auto",
				"params": map[string]any{},
			},
		},
	}
	dst := &monitorModel{Type: types.StringValue("metric")}

	diags := applyMonitorToModel(mon, dst)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Nil(t, dst.ParamsMetric.Column, "server-derived column must be dropped when user did not set one")
	require.True(t, dst.ParamsMetric.Resolution.IsNull(), "server-defaulted resolution must stay null")
	require.True(t, dst.ParamsMetric.AbsentPoints.IsNull(), "server-defaulted absent_points must stay null")
	require.True(t, dst.ParamsMetric.NumEvalPoints.IsNull(), "server-defaulted num_eval_points must stay null")
}

// User sets column.name = "avg(x)". If the backend were to normalize it
// (e.g. "avg($x)"), state must keep the user's input form to avoid a
// post-apply consistency error. Same contract as preserveQuery.
func TestApplyMetricMonitor_columnNamePreservedOnNormalization(t *testing.T) {
	mon := &generated.Monitor{
		ID:     104,
		Name:   "norm-col",
		Type:   generated.MonitorTypeMetric,
		Status: generated.Active,
		Params: map[string]any{
			"query":   "avg($x)",
			"metrics": []any{map[string]any{"name": "m", "alias": "$x"}},
			"column":  map[string]any{"name": "avg($x)", "unit": "ms"},
			"detector": map[string]any{
				"type":   "auto",
				"params": map[string]any{},
			},
		},
	}
	dst := &monitorModel{
		Type: types.StringValue("metric"),
		ParamsMetric: &metricParamsModel{
			Column: &columnModel{
				Name: types.StringValue("avg(x)"),
				Unit: types.StringValue("ms"),
			},
		},
	}

	diags := applyMonitorToModel(mon, dst)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, "avg(x)", dst.ParamsMetric.Column.Name.ValueString(), "user-set column.name must be preserved verbatim")
	require.Equal(t, "ms", dst.ParamsMetric.Column.Unit.ValueString())
}

func TestApplyMetricMonitor_manualDetectorWithRecovery(t *testing.T) {
	mon := &generated.Monitor{
		ID:     102,
		Name:   "thresh",
		Type:   generated.MonitorTypeMetric,
		Status: generated.Active,
		Params: map[string]any{
			"query":   "sum($x)",
			"metrics": []any{map[string]any{"name": "m", "alias": "$x"}},
			"detector": map[string]any{
				"type": "manual",
				"params": map[string]any{
					"minValue": 1,
					"maxValue": 100,
					"recovery": map[string]any{"minValue": 10, "maxValue": 90},
				},
			},
		},
	}
	dst := &monitorModel{Type: types.StringValue("metric")}

	diags := applyMonitorToModel(mon, dst)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.NotNil(t, dst.ParamsMetric.Detector.Manual)
	require.Equal(t, float64(1), dst.ParamsMetric.Detector.Manual.MinValue.ValueFloat64())
	require.NotNil(t, dst.ParamsMetric.Detector.Manual.Recovery)
	require.Equal(t, float64(10), dst.ParamsMetric.Detector.Manual.Recovery.MinValue.ValueFloat64())
}

// --- ValidateConfig ---

func TestValidateExactlyOneParamsBlock_errorMissing(t *testing.T) {
	cfg := &monitorModel{Type: types.StringValue("error")}
	var diags diag.Diagnostics
	validateExactlyOneParamsBlock(cfg, "error", &diags)
	require.True(t, diags.HasError(), "params_error required for type=error")
}

func TestValidateExactlyOneParamsBlock_wrongBlock(t *testing.T) {
	cfg := &monitorModel{
		Type:        types.StringValue("error"),
		ParamsError: &errorParamsModel{},
		ParamsMetric: &metricParamsModel{
			Detector: &detectorModel{Auto: &autoDetectorModel{}},
		},
	}
	var diags diag.Diagnostics
	validateExactlyOneParamsBlock(cfg, "error", &diags)
	require.True(t, diags.HasError(), "params_metric must not be set when type=error")
}

func TestValidateDetector(t *testing.T) {
	tests := []struct {
		name      string
		detector  *detectorModel
		wantError bool
	}{
		{"both manual and auto set", &detectorModel{Manual: &manualDetectorModel{}, Auto: &autoDetectorModel{}}, true},
		{"neither set", &detectorModel{}, true},
		{"auto only", &detectorModel{Auto: &autoDetectorModel{}}, false},
		{"manual only", &detectorModel{Manual: &manualDetectorModel{}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			validateDetector(&metricParamsModel{Detector: tc.detector}, &diags)
			require.Equal(t, tc.wantError, diags.HasError())
		})
	}
}
