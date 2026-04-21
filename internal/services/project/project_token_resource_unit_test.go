package project

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

func TestProjectTokenToModel_fullPayload(t *testing.T) {
	token := &generated.ProjectToken{
		ID:        42,
		ProjectID: 7,
		Name:      runtime.Ptr("ci-ingest"),
		Token:     "secret123",
		Dsn:       runtime.Ptr("http://secret123@localhost:14318/7"),
	}
	m := projectTokenModel{ProjectID: types.StringValue("7")}

	projectTokenToModel(token, &m)

	require.Equal(t, types.StringValue("42"), m.ID)
	require.Equal(t, types.StringValue("7"), m.ProjectID, "ProjectID must not be modified by the mapper")
	require.Equal(t, types.StringValue("ci-ingest"), m.Name)
	require.Equal(t, types.StringValue("secret123"), m.Token)
	require.Equal(t, types.StringValue("http://secret123@localhost:14318/7"), m.DSN)
}

func TestProjectTokenToModel_doesNotSetProjectID(t *testing.T) {
	token := &generated.ProjectToken{
		ID:        42,
		ProjectID: 7,
		Token:     "secret123",
	}
	m := projectTokenModel{ProjectID: types.StringValue("99")}

	projectTokenToModel(token, &m)

	require.Equal(t, types.StringValue("99"), m.ProjectID,
		"mapper must not overwrite ProjectID — callers own that field")
}

func TestProjectTokenToModel_nilOptionalFields(t *testing.T) {
	token := &generated.ProjectToken{
		ID:        42,
		ProjectID: 7,
		Token:     "secret123",
	}
	var m projectTokenModel

	projectTokenToModel(token, &m)

	require.Equal(t, types.StringValue("42"), m.ID)
	require.Equal(t, types.StringValue("secret123"), m.Token)
	require.True(t, m.Name.IsNull())
	require.True(t, m.DSN.IsNull())
}

func TestProjectTokenToModel_emptyNameTreatedAsNull(t *testing.T) {
	token := &generated.ProjectToken{
		ID:        42,
		ProjectID: 7,
		Token:     "secret123",
		Name:      runtime.Ptr(""),
	}
	var m projectTokenModel

	projectTokenToModel(token, &m)

	require.True(t, m.Name.IsNull(),
		"empty string name from API must map to null to avoid drift when user omits name")
}

func TestProjectTokenImportState_rejectsInvalidTokenID(t *testing.T) {
	var resp resource.ImportStateResponse

	(&ProjectTokenResource{}).ImportState(context.Background(), resource.ImportStateRequest{ID: "7:not-a-token"}, &resp)

	require.True(t, resp.Diagnostics.HasError())
	require.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "invalid token_id in import ID")
}
