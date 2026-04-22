package monitor_test

import (
	"strings"
	"testing"
)

func TestAccErrorMonitorConfigWithChannels_usesWebhookChannelResource(t *testing.T) {
	config := testAccErrorMonitorConfigWithChannels(
		"acc-org",
		"acc-project",
		"acc-monitor",
		"https://example.com/hooks/alert",
		"[uptrace_webhook_channel.test.id]",
	)

	if strings.Contains(config, `resource "uptrace_notification_channel" "test"`) {
		t.Fatalf("monitor acc config still references deleted notification channel resource:\n%s", config)
	}
	if !strings.Contains(config, `resource "uptrace_webhook_channel" "test"`) {
		t.Fatalf("monitor acc config must create a typed webhook channel resource:\n%s", config)
	}
	if !strings.Contains(config, `channel_ids = [uptrace_webhook_channel.test.id]`) {
		t.Fatalf("monitor acc config must wire webhook channel IDs into channel_ids:\n%s", config)
	}
}
