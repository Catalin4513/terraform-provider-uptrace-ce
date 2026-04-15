package org

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

func TestOrgToModel_basic(t *testing.T) {
	org := &generated.Org{ID: 42, Name: "Acme", Budget: runtime.Ptr(100.0)}
	var m orgModel

	orgToModel(org, &m)

	require.Equal(t, types.StringValue("42"), m.ID)
	require.Equal(t, types.StringValue("Acme"), m.Name)
	require.Equal(t, types.Float64Value(100.0), m.Budget)
}

func TestOrgToModel_nilBudget(t *testing.T) {
	model := orgModel{
		ID:     types.StringValue("41"),
		Name:   types.StringValue("old-name"),
		Budget: types.Float64Value(250),
	}

	orgToModel(&generated.Org{ID: 42, Name: "new-name"}, &model)

	require.Equal(t, types.StringValue("42"), model.ID)
	require.Equal(t, types.StringValue("new-name"), model.Name)
	require.True(t, model.Budget.IsNull(), "budget should be null when API omits it")
}

func TestParseOrgID_valid(t *testing.T) {
	id, err := parseOrgID("42")
	require.NoError(t, err)
	require.Equal(t, uint64(42), id)
}

func TestParseOrgID_invalid(t *testing.T) {
	_, err := parseOrgID("not-a-number")
	require.Error(t, err)
}

func TestParseOrgID_negative(t *testing.T) {
	_, err := parseOrgID("-1")
	require.Error(t, err)
}
