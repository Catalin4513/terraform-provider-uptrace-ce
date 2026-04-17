package monitor_test

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

func testAccErrorMonitorConfigBasic(orgName, projectName, monitorName string) string {
	return fmt.Sprintf(`
resource "uptrace_org" "test" {
  name = %q
}

resource "uptrace_project" "test" {
  org_id = uptrace_org.test.id
  name   = %q
}

resource "uptrace_error_monitor" "test" {
  project_id = uptrace_project.test.id
  name       = %q

  params = {
    query = "sum($logs) | where _system in (\"log:error\", \"log:fatal\")"
    metrics = [
      { name = "uptrace_tracing_logs", alias = "$logs" }
    ]
  }
}
`, orgName, projectName, monitorName)
}

func testAccCheckErrorMonitorDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := testutil.TestAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "uptrace_error_monitor" {
				continue
			}
			monitorID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid monitor ID %q: %w", rs.Primary.ID, err)
			}
			projectID, err := strconv.ParseUint(rs.Primary.Attributes["project_id"], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid project ID %q: %w", rs.Primary.Attributes["project_id"], err)
			}
			_, err = c.API.GetMonitor(context.Background(), &generated.GetMonitorRequestOptions{
				PathParams: &generated.GetMonitorPath{
					ProjectID: uint32(projectID),
					MonitorID: monitorID,
				},
			})
			if err == nil {
				return fmt.Errorf("error monitor %s still exists after destroy", rs.Primary.ID)
			}
			if !client.IsNotFound(err) && !client.IsForbidden(err) {
				return fmt.Errorf("checking error monitor %s after destroy: %w", rs.Primary.ID, err)
			}
		}
		return nil
	}
}

func TestAccErrorMonitor_basic(t *testing.T) {
	const (
		orgName     = "acc-error-monitor-org"
		projectName = "acc-error-monitor-project"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckErrorMonitorDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccErrorMonitorConfigBasic(orgName, projectName, "acc-em-basic"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_error_monitor.test", "id"),
					resource.TestCheckResourceAttr("uptrace_error_monitor.test", "name", "acc-em-basic"),
					resource.TestCheckResourceAttr("uptrace_error_monitor.test", "notify_everyone_by_email", "false"),
					resource.TestCheckResourceAttr("uptrace_error_monitor.test", "trend_agg_func", "sum"),
					resource.TestCheckResourceAttr("uptrace_error_monitor.test", "trend_sensitivity", "medium"),
					resource.TestCheckResourceAttr("uptrace_error_monitor.test", "params.metrics.#", "1"),
					resource.TestCheckResourceAttr("uptrace_error_monitor.test", "params.metrics.0.name", "uptrace_tracing_logs"),
					resource.TestCheckResourceAttr("uptrace_error_monitor.test", "params.metrics.0.alias", "$logs"),
					resource.TestCheckResourceAttrSet("uptrace_error_monitor.test", "status"),
				),
			},
			{
				Config:   testAccErrorMonitorConfigBasic(orgName, projectName, "acc-em-basic"),
				PlanOnly: true,
			},
			{
				Config: testAccErrorMonitorConfigBasic(orgName, projectName, "acc-em-basic-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptrace_error_monitor.test", "name", "acc-em-basic-renamed"),
				),
			},
			{
				ResourceName:      "uptrace_error_monitor.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The backend normalizes MQL (e.g. $logs → $logs{}, all.type → all.type::str),
				// so the imported query can differ from the user's input form even though
				// they're semantically identical.
				ImportStateVerifyIgnore: []string{"params"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["uptrace_error_monitor.test"]
					if !ok {
						return "", fmt.Errorf("uptrace_error_monitor.test not found in state")
					}
					return fmt.Sprintf("%s:%s", rs.Primary.Attributes["project_id"], rs.Primary.ID), nil
				},
			},
		},
	})
}

// testAccErrorMonitorConfigWithChannels wires channel_ids to a
// uptrace_notification_channel. The channel uses lifecycle.ignore_changes on
// monitor_ids because the backend populates that attr from the reverse link
// and the current notification_channel resource surfaces it as drift.
func testAccErrorMonitorConfigWithChannels(orgName, projectName, monitorName, webhookURL, channelIDsExpr string) string {
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
  name       = "acc-em-chan-webhook"
  type       = "webhook"
  priorities = ["high"]

  webhook {
    url = %q
  }

  lifecycle {
    ignore_changes = [monitor_ids]
  }
}

resource "uptrace_error_monitor" "test" {
  project_id  = uptrace_project.test.id
  name        = %q
  channel_ids = %s

  params = {
    query = "sum($logs) | where _system in (\"log:error\", \"log:fatal\")"
    metrics = [
      { name = "uptrace_tracing_logs", alias = "$logs" }
    ]
  }
}
`, orgName, projectName, webhookURL, monitorName, channelIDsExpr)
}

func TestAccErrorMonitor_channelIDsClearOnRemove(t *testing.T) {
	const (
		orgName     = "acc-em-chan-clear-org"
		projectName = "acc-em-chan-clear-project"
		webhookURL  = "https://example.com/hooks/alert"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckErrorMonitorDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccErrorMonitorConfigWithChannels(orgName, projectName, "acc-em-chan-clear", webhookURL,
					"[uptrace_notification_channel.test.id]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptrace_error_monitor.test", "channel_ids.#", "1"),
					resource.TestCheckResourceAttrPair(
						"uptrace_error_monitor.test", "channel_ids.0",
						"uptrace_notification_channel.test", "id",
					),
				),
			},
			{
				Config: testAccErrorMonitorConfigWithChannels(orgName, projectName, "acc-em-chan-clear", webhookURL, "null"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("uptrace_error_monitor.test", "channel_ids.#"),
				),
			},
			{
				Config:   testAccErrorMonitorConfigWithChannels(orgName, projectName, "acc-em-chan-clear", webhookURL, "null"),
				PlanOnly: true,
			},
		},
	})
}

func TestAccErrorMonitor_disappearsOutOfBand(t *testing.T) {
	const (
		orgName     = "acc-em-disappear-org"
		projectName = "acc-em-disappear-project"
	)
	config := testAccErrorMonitorConfigBasic(orgName, projectName, "acc-em-disappear")
	var monitorID, projectID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckErrorMonitorDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_error_monitor.test", "id"),
					testutil.CaptureAttr("uptrace_error_monitor.test", "id", &monitorID),
					testutil.CaptureAttr("uptrace_error_monitor.test", "project_id", &projectID),
				),
			},
			{
				PreConfig: func() {
					deleteErrorMonitorOutOfBand(t, projectID, monitorID)
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_error_monitor.test", "id"),
					resource.TestCheckResourceAttr("uptrace_error_monitor.test", "name", "acc-em-disappear"),
				),
			},
		},
	})
}

func deleteErrorMonitorOutOfBand(t *testing.T, projectIDStr, monitorIDStr string) {
	t.Helper()
	if projectIDStr == "" || monitorIDStr == "" {
		t.Fatal("project ID or monitor ID was not captured before out-of-band delete")
	}
	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		t.Fatalf("invalid project ID %q: %v", projectIDStr, err)
	}
	monitorID, err := strconv.ParseInt(monitorIDStr, 10, 64)
	if err != nil {
		t.Fatalf("invalid monitor ID %q: %v", monitorIDStr, err)
	}

	c := testutil.TestAccClient(t)
	_, err = c.API.DeleteMonitor(context.Background(), &generated.DeleteMonitorRequestOptions{
		PathParams: &generated.DeleteMonitorPath{
			ProjectID: uint32(projectID),
			MonitorID: monitorID,
		},
	})
	if err != nil && !client.IsNotFound(err) && !client.IsForbidden(err) {
		t.Fatalf("delete error monitor %d out-of-band: %v", monitorID, err)
	}
}
