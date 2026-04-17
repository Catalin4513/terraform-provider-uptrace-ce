package notificationchannel_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/client"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/testutil"
)

func testAccNotificationChannelConfigBasic(orgName, projectName, channelName, webhookURL string) string {
	return fmt.Sprintf(`
resource "uptrace_org" "test" {
  name = %q
}

resource "uptrace_project" "test" {
  org_id = uptrace_org.test.id
  name   = %q
}

resource "uptrace_notification_channel" "test" {
  project_id = uptrace_project.test.id
  name       = %q
  type       = "webhook"
  priorities = ["high"]

  webhook {
    url = %q
  }
}
`, orgName, projectName, channelName, webhookURL)
}

func testAccNotificationChannelConfigSlackWebhook(orgName, projectName, channelName, webhookURL string) string {
	return fmt.Sprintf(`
resource "uptrace_org" "test" {
  name = %q
}

resource "uptrace_project" "test" {
  org_id = uptrace_org.test.id
  name   = %q
}

resource "uptrace_notification_channel" "test" {
  project_id = uptrace_project.test.id
  name       = %q
  type       = "slack"
  priorities = ["high"]

  slack {
    auth_method = "webhook"
    webhook_url = %q
  }
}
`, orgName, projectName, channelName, webhookURL)
}

func testAccNotificationChannelConfigSlackToken(orgName, projectName, channelName, token, slackChannel string) string {
	return fmt.Sprintf(`
resource "uptrace_org" "test" {
  name = %q
}

resource "uptrace_project" "test" {
  org_id = uptrace_org.test.id
  name   = %q
}

resource "uptrace_notification_channel" "test" {
  project_id = uptrace_project.test.id
  name       = %q
  type       = "slack"
  priorities = ["high", "medium"]

  slack {
    auth_method = "token"
    token       = %q
    channel     = %q
  }
}
`, orgName, projectName, channelName, token, slackChannel)
}

func testAccCheckNotificationChannelDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := testutil.TestAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "uptrace_notification_channel" {
				continue
			}
			channelID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid channel ID %q: %w", rs.Primary.ID, err)
			}
			projectID, err := strconv.ParseUint(rs.Primary.Attributes["project_id"], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid project ID %q: %w", rs.Primary.Attributes["project_id"], err)
			}
			_, err = c.API.GetNotificationChannel(context.Background(), &generated.GetNotificationChannelRequestOptions{
				PathParams: &generated.GetNotificationChannelPath{
					ProjectID: uint32(projectID),
					ChannelID: channelID,
				},
			})
			if err == nil {
				return fmt.Errorf("notification channel %s still exists after destroy", rs.Primary.ID)
			}
			if !client.IsNotFound(err) && !client.IsForbidden(err) {
				return fmt.Errorf("checking notification channel %s after destroy: %w", rs.Primary.ID, err)
			}
		}
		return nil
	}
}

func TestAccNotificationChannel_basic(t *testing.T) {
	const (
		orgName     = "acc-channel-basic-org"
		projectName = "acc-channel-basic-project"
		webhookURL  = "https://example.com/hooks/alert"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNotificationChannelDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationChannelConfigBasic(orgName, projectName, "acc-basic", webhookURL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_notification_channel.test", "id"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "type", "webhook"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "webhook.url", webhookURL),
				),
			},
			{
				Config:   testAccNotificationChannelConfigBasic(orgName, projectName, "acc-basic", webhookURL),
				PlanOnly: true,
			},
			{
				Config: testAccNotificationChannelConfigBasic(orgName, projectName, "acc-basic-renamed", webhookURL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "name", "acc-basic-renamed"),
				),
			},
			{
				ResourceName:            "uptrace_notification_channel.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"webhook"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["uptrace_notification_channel.test"]
					if !ok {
						return "", fmt.Errorf("uptrace_notification_channel.test not found in state")
					}
					return fmt.Sprintf("%s:%s", rs.Primary.Attributes["project_id"], rs.Primary.ID), nil
				},
			},
		},
	})
}

func TestAccNotificationChannel_slackWebhook(t *testing.T) {
	const (
		orgName     = "acc-channel-slack-org"
		projectName = "acc-channel-slack-project"
		webhookURL  = "https://hooks.slack.com/services/T00/B00/XXXX"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNotificationChannelDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationChannelConfigSlackWebhook(orgName, projectName, "acc-slack-webhook", webhookURL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_notification_channel.test", "id"),
					resource.TestCheckResourceAttrSet("uptrace_notification_channel.test", "project_id"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "name", "acc-slack-webhook"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "type", "slack"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "match_all", "true"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "priorities.#", "1"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "priorities.0", "high"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "slack.auth_method", "webhook"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "slack.webhook_url", webhookURL),
					resource.TestCheckResourceAttrSet("uptrace_notification_channel.test", "status"),
				),
			},
			{
				Config:   testAccNotificationChannelConfigSlackWebhook(orgName, projectName, "acc-slack-webhook", webhookURL),
				PlanOnly: true,
			},
			{
				Config: testAccNotificationChannelConfigSlackWebhook(orgName, projectName, "acc-slack-webhook-renamed", webhookURL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "name", "acc-slack-webhook-renamed"),
				),
			},
			{
				ResourceName:      "uptrace_notification_channel.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Sensitive fields are not returned by the API on read, so they can't
				// round-trip through import verification.
				ImportStateVerifyIgnore: []string{"slack"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["uptrace_notification_channel.test"]
					if !ok {
						return "", fmt.Errorf("uptrace_notification_channel.test not found in state")
					}
					return fmt.Sprintf("%s:%s", rs.Primary.Attributes["project_id"], rs.Primary.ID), nil
				},
			},
		},
	})
}

func TestAccNotificationChannel_slackToken(t *testing.T) {
	const (
		orgName      = "acc-channel-slack-token-org"
		projectName  = "acc-channel-slack-token-project"
		token        = "slack-bot-token-placeholder"
		slackChannel = "#alerts"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNotificationChannelDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationChannelConfigSlackToken(orgName, projectName, "acc-slack-token", token, slackChannel),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "type", "slack"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "priorities.#", "2"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "slack.auth_method", "token"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "slack.token", token),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "slack.channel", slackChannel),
				),
			},
			{
				Config:   testAccNotificationChannelConfigSlackToken(orgName, projectName, "acc-slack-token", token, slackChannel),
				PlanOnly: true,
			},
			{
				Config: testAccNotificationChannelConfigSlackToken(orgName, projectName, "acc-slack-token-renamed", token, slackChannel),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "name", "acc-slack-token-renamed"),
				),
			},
		},
	})
}

func TestAccNotificationChannel_disappearsOutOfBand(t *testing.T) {
	const (
		orgName     = "acc-disappear-channel-org"
		projectName = "acc-disappear-channel-project"
		webhookURL  = "https://example.com/hooks/disappearing"
	)
	config := testAccNotificationChannelConfigBasic(orgName, projectName, "acc-disappear-channel", webhookURL)
	var channelID, projectID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNotificationChannelDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_notification_channel.test", "id"),
					testutil.CaptureAttr("uptrace_notification_channel.test", "id", &channelID),
					testutil.CaptureAttr("uptrace_notification_channel.test", "project_id", &projectID),
				),
			},
			{
				PreConfig: func() {
					deleteNotificationChannelOutOfBand(t, projectID, channelID)
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_notification_channel.test", "id"),
					resource.TestCheckResourceAttr("uptrace_notification_channel.test", "name", "acc-disappear-channel"),
				),
			},
		},
	})
}

func deleteNotificationChannelOutOfBand(t *testing.T, projectIDStr, channelIDStr string) {
	t.Helper()
	if projectIDStr == "" || channelIDStr == "" {
		t.Fatal("project ID or channel ID was not captured before out-of-band delete")
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		t.Fatalf("invalid project ID %q: %v", projectIDStr, err)
	}
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		t.Fatalf("invalid channel ID %q: %v", channelIDStr, err)
	}

	c := testutil.TestAccClient(t)
	_, err = c.API.DeleteNotificationChannel(context.Background(), &generated.DeleteNotificationChannelRequestOptions{
		PathParams: &generated.DeleteNotificationChannelPath{
			ProjectID: uint32(projectID),
			ChannelID: channelID,
		},
	})
	if err != nil && !client.IsNotFound(err) && !client.IsForbidden(err) {
		t.Fatalf("delete notification channel %d out-of-band: %v", channelID, err)
	}
}
