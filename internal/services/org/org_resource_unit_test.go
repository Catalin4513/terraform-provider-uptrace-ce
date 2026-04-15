package org

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

func TestOrgToModelClearsBudgetWhenAPIOmitsIt(t *testing.T) {
	model := orgModel{
		ID:     types.StringValue("41"),
		Name:   types.StringValue("old-name"),
		Budget: types.Float64Value(250),
	}

	orgToModel(&generated.Org{
		ID:   42,
		Name: "new-name",
	}, &model)

	require.Equal(t, types.StringValue("42"), model.ID)
	require.Equal(t, types.StringValue("new-name"), model.Name)
	require.True(t, model.Budget.IsNull())
}
