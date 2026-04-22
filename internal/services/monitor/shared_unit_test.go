package monitor

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/require"
)

func TestSharedAttributes_channelIDsDescriptionUsesTypedChannelResource(t *testing.T) {
	attr, ok := sharedAttributes()["channel_ids"].(schema.SetAttribute)
	require.True(t, ok, "channel_ids must stay a set attribute")

	require.NotContains(t, attr.Description, "uptrace_notification_channel.id")
	require.Contains(t, attr.Description, "uptrace_webhook_channel.id")
}
