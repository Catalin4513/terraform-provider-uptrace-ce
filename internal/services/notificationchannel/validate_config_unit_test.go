package notificationchannel

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestValidateSlackBlock_tokenRequiresChannel(t *testing.T) {
	m := &slackModel{
		AuthMethod: types.StringValue("token"),
		Token:      types.StringValue("xoxb-secret"),
		Channel:    types.StringNull(),
		WebhookURL: types.StringNull(),
	}

	var diags diag.Diagnostics
	validateSlackBlock(m, &diags)

	require.True(t, diags.HasError())
	require.Contains(t, diags.Errors()[0].Detail(), "slack.channel must be set")
}
