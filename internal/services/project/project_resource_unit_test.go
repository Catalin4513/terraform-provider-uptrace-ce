package project

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"
	str2duration "github.com/xhit/go-str2duration/v2"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

// durationMs parses a duration string with the same library the resource
// uses and returns it as float64 milliseconds — the wire representation the
// server expects.
func durationMs(t *testing.T, s string) float64 {
	t.Helper()
	d, err := str2duration.ParseDuration(s)
	require.NoError(t, err)
	return float64(d.Milliseconds())
}

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

	require.Equal(t, "7", m.ID.ValueString())
	require.Equal(t, "42", m.OrgID.ValueString())
	require.Equal(t, "api", m.Name.ValueString())
	require.True(t, m.GroupByEnv.ValueBool())
	require.False(t, m.GroupFuncsByService.ValueBool())
	require.Equal(t, "v1.33.0", m.SemconvVersion.ValueString())
	require.True(t, m.DisplayLogSeverity.ValueBool())
	require.False(t, m.CountDistinct.ValueBool())
}

func TestProjectToModel_nilOrgID(t *testing.T) {
	m := projectModel{OrgID: types.StringValue("42")}
	projectToModel(&generated.Project{ID: 1, Name: "x"}, &m)
	require.True(t, m.OrgID.IsNull())
}

func TestProjectToModel_retentionAndTimeRangeNotOverwritten(t *testing.T) {
	m := projectModel{
		SpanRetention:   types.StringValue("720h"),
		MetricRetention: types.StringValue("720h"),
		SpanTimeRange:   types.StringValue("24h"),
	}
	apiResp := &generated.Project{
		ID:              1,
		Name:            "api",
		SpanRetention:   runtime.Ptr[float64](1_209_600_000), // 336h — the CE override
		MetricRetention: runtime.Ptr[float64](1_209_600_000),
		SpanTimeRange:   runtime.Ptr[float64](0),
	}

	projectToModel(apiResp, &m)

	require.Equal(t, "720h", m.SpanRetention.ValueString(),
		"retention must not be overwritten by the API response")
	require.Equal(t, "720h", m.MetricRetention.ValueString())
	require.Equal(t, "24h", m.SpanTimeRange.ValueString())
}

func TestProjectRequestBody_retentionAndTimeRangeConvertedToMillis(t *testing.T) {
	m := &projectModel{
		Name:           types.StringValue("api"),
		SpanRetention:  types.StringValue("30d"), // str2duration-only unit
		SpanTimeRange:  types.StringValue("24h"),
		EventRetention: types.StringValue("0s"),
	}

	body := projectRequestBody(m)

	require.NotNil(t, body.SpanRetention)
	require.Equal(t, durationMs(t, "30d"), *body.SpanRetention)
	require.NotNil(t, body.SpanTimeRange)
	require.Equal(t, durationMs(t, "24h"), *body.SpanTimeRange)
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
	require.Nil(t, body.SpanRetention)
	require.Nil(t, body.MetricRetention)
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
