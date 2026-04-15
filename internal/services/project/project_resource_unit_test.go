package project

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

func TestProjectToModel_fullPayload(t *testing.T) {
	semconv := generated.V1330
	p := &generated.Project{
		ID:                  7,
		OrgID:               runtime.Ptr[uint64](42),
		Name:                "api",
		GroupByEnv:          runtime.Ptr(true),
		GroupFuncsByService: runtime.Ptr(false),
		SemconvVersion:      &semconv,
		DisplayLogSeverity:  runtime.Ptr(true),
		CountDistinct:       runtime.Ptr(false),
	}
	var m projectModel

	projectToModel(p, &m)

	require.Equal(t, types.StringValue("7"), m.ID)
	require.Equal(t, types.StringValue("42"), m.OrgID)
	require.Equal(t, types.StringValue("api"), m.Name)
	require.Equal(t, types.BoolValue(true), m.GroupByEnv)
	require.Equal(t, types.BoolValue(false), m.GroupFuncsByService)
	require.Equal(t, types.StringValue("v1.33.0"), m.SemconvVersion)
	require.Equal(t, types.BoolValue(true), m.DisplayLogSeverity)
	require.Equal(t, types.BoolValue(false), m.CountDistinct)
}

func TestProjectToModel_nilOrgID(t *testing.T) {
	m := projectModel{OrgID: types.StringValue("42")}
	projectToModel(&generated.Project{ID: 1, Name: "x"}, &m)
	require.True(t, m.OrgID.IsNull())
}

func TestProjectToModel_retentionAndTimeRangeNotOverwritten(t *testing.T) {
	m := projectModel{
		SpanRetention:   types.Float64Value(2_592_000_000), // 720h in ms
		MetricRetention: types.Float64Value(2_592_000_000),
		SpanTimeRange:   types.Float64Value(86_400_000), // 24h in ms
	}
	apiResp := &generated.Project{
		ID:              1,
		Name:            "api",
		SpanRetention:   runtime.Ptr[float64](1_209_600_000), // 336h — the CE override
		MetricRetention: runtime.Ptr[float64](1_209_600_000),
		SpanTimeRange:   runtime.Ptr[float64](0),
	}

	projectToModel(apiResp, &m)

	require.Equal(t, types.Float64Value(2_592_000_000), m.SpanRetention,
		"retention must not be overwritten by the API response")
	require.Equal(t, types.Float64Value(2_592_000_000), m.MetricRetention)
	require.Equal(t, types.Float64Value(86_400_000), m.SpanTimeRange)
}

func TestProjectRequestBody_retentionAndTimeRangePassedThrough(t *testing.T) {
	m := &projectModel{
		Name:           types.StringValue("api"),
		SpanRetention:  types.Float64Value(2_592_000_000),
		SpanTimeRange:  types.Float64Value(86_400_000),
		EventRetention: types.Float64Value(0),
	}

	body := projectRequestBody(m)

	require.NotNil(t, body.SpanRetention)
	require.Equal(t, float64(2_592_000_000), *body.SpanRetention)
	require.NotNil(t, body.SpanTimeRange)
	require.Equal(t, float64(86_400_000), *body.SpanTimeRange)
	// Explicit zero is passed through as the "use default" sentinel.
	require.NotNil(t, body.EventRetention)
	require.Equal(t, float64(0), *body.EventRetention)
	// Unset field is omitted.
	require.Nil(t, body.LogRetention)
}

func TestProjectRequestBody_unsetFieldsAreOmitted(t *testing.T) {
	m := &projectModel{
		Name:       types.StringValue("api"),
		GroupByEnv: types.BoolValue(true),
	}

	body := projectRequestBody(m)

	require.Equal(t, "api", body.Name)
	require.NotNil(t, body.GroupByEnv)
	require.True(t, *body.GroupByEnv)
	require.Nil(t, body.GroupFuncsByService)
	require.Nil(t, body.SemconvVersion)
	require.Nil(t, body.DisplayLogSeverity)
}

func TestProjectRequestBody_semconvEnumIsPassedThrough(t *testing.T) {
	m := &projectModel{
		Name:           types.StringValue("api"),
		SemconvVersion: types.StringValue("v1.25.0"),
	}

	body := projectRequestBody(m)

	require.NotNil(t, body.SemconvVersion)
	require.Equal(t, generated.ProjectCreateRequestSemconvVersionV1250, *body.SemconvVersion)
}

func TestParseProjectID_valid(t *testing.T) {
	id, err := parseProjectID("123")
	require.NoError(t, err)
	require.Equal(t, int64(123), id)
}

func TestParseProjectID_invalid(t *testing.T) {
	_, err := parseProjectID("abc")
	require.Error(t, err)
}
