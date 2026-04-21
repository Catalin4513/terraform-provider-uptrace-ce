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

resource "uptrace_monitor" "test" {
  project_id = uptrace_project.test.id
  name       = %q
  type       = "error"

  params_error = {
    query = "sum($logs) | where _system in (\"log:error\", \"log:fatal\")"
    metrics = [
      { name = "uptrace_tracing_logs", alias = "$logs" }
    ]
  }
}
`, orgName, projectName, monitorName)
}

func testAccCheckMonitorDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := testutil.TestAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "uptrace_monitor" {
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
				return fmt.Errorf("monitor %s still exists after destroy", rs.Primary.ID)
			}
			if !client.IsNotFound(err) && !client.IsForbidden(err) {
				return fmt.Errorf("checking monitor %s after destroy: %w", rs.Primary.ID, err)
			}
		}
		return nil
	}
}

func TestAccMonitor_errorBasic(t *testing.T) {
	const (
		orgName     = "acc-monitor-err-org"
		projectName = "acc-monitor-err-project"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccErrorMonitorConfigBasic(orgName, projectName, "acc-em-basic"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_monitor.test", "id"),
					resource.TestCheckResourceAttr("uptrace_monitor.test", "type", "error"),
					resource.TestCheckResourceAttr("uptrace_monitor.test", "name", "acc-em-basic"),
					resource.TestCheckResourceAttr("uptrace_monitor.test", "trend_agg_func", "sum"),
					resource.TestCheckResourceAttr("uptrace_monitor.test", "trend_sensitivity", "medium"),
					resource.TestCheckResourceAttr("uptrace_monitor.test", "params_error.metrics.#", "1"),
					resource.TestCheckResourceAttr("uptrace_monitor.test", "params_error.metrics.0.name", "uptrace_tracing_logs"),
					resource.TestCheckResourceAttr("uptrace_monitor.test", "params_error.metrics.0.alias", "$logs"),
					resource.TestCheckResourceAttrSet("uptrace_monitor.test", "status"),
				),
			},
			{
				Config:   testAccErrorMonitorConfigBasic(orgName, projectName, "acc-em-basic"),
				PlanOnly: true,
			},
			{
				Config: testAccErrorMonitorConfigBasic(orgName, projectName, "acc-em-basic-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptrace_monitor.test", "name", "acc-em-basic-renamed"),
				),
			},
			{
				ResourceName:      "uptrace_monitor.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The backend normalizes MQL (e.g. $logs → $logs{}, all.type → all.type::str),
				// so the imported query can differ from the user's input form even though
				// they're semantically identical.
				ImportStateVerifyIgnore: []string{"params_error"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["uptrace_monitor.test"]
					if !ok {
						return "", fmt.Errorf("uptrace_monitor.test not found in state")
					}
					return fmt.Sprintf("%s:%s", rs.Primary.Attributes["project_id"], rs.Primary.ID), nil
				},
			},
		},
	})
}

func testAccMonitorConfigErrorWithChannels(orgName, projectName, monitorName, webhookURL, channelIDsExpr string) string {
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

resource "uptrace_monitor" "test" {
  project_id  = uptrace_project.test.id
  name        = %q
  type        = "error"
  channel_ids = %s

  params_error = {
    query = "sum($logs) | where _system in (\"log:error\", \"log:fatal\")"
    metrics = [
      { name = "uptrace_tracing_logs", alias = "$logs" }
    ]
  }
}
`, orgName, projectName, webhookURL, monitorName, channelIDsExpr)
}

func TestAccMonitor_errorChannelIDsClearOnRemove(t *testing.T) {
	const (
		orgName     = "acc-em-chan-clear-org"
		projectName = "acc-em-chan-clear-project"
		webhookURL  = "https://example.com/hooks/alert"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorConfigErrorWithChannels(orgName, projectName, "acc-em-chan-clear", webhookURL,
					"[uptrace_notification_channel.test.id]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptrace_monitor.test", "channel_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(
						"uptrace_monitor.test", "channel_ids.*",
						"uptrace_notification_channel.test", "id",
					),
				),
			},
			{
				Config: testAccMonitorConfigErrorWithChannels(orgName, projectName, "acc-em-chan-clear", webhookURL, "null"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("uptrace_monitor.test", "channel_ids.#"),
				),
			},
			{
				Config:   testAccMonitorConfigErrorWithChannels(orgName, projectName, "acc-em-chan-clear", webhookURL, "null"),
				PlanOnly: true,
			},
		},
	})
}

func TestAccMonitor_errorDisappearsOutOfBand(t *testing.T) {
	const (
		orgName     = "acc-em-disappear-org"
		projectName = "acc-em-disappear-project"
	)
	config := testAccErrorMonitorConfigBasic(orgName, projectName, "acc-em-disappear")
	var monitorID, projectID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_monitor.test", "id"),
					testutil.CaptureAttr("uptrace_monitor.test", "id", &monitorID),
					testutil.CaptureAttr("uptrace_monitor.test", "project_id", &projectID),
				),
			},
			{
				PreConfig: func() {
					deleteMonitorOutOfBand(t, projectID, monitorID)
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_monitor.test", "id"),
					resource.TestCheckResourceAttr("uptrace_monitor.test", "name", "acc-em-disappear"),
				),
			},
		},
	})
}

func testAccMetricMonitorConfigAuto(orgName, projectName, monitorName string) string {
	return fmt.Sprintf(`
resource "uptrace_org" "test" {
  name = %q
}

resource "uptrace_project" "test" {
  org_id = uptrace_org.test.id
  name   = %q
}

resource "uptrace_monitor" "test" {
  project_id = uptrace_project.test.id
  name       = %q
  type       = "metric"

  params_metric = {
    query = "avg($http_duration)"
    metrics = [
      { name = "uptrace_tracing_spans", alias = "$http_duration" }
    ]
    column = {
      name = "avg($http_duration)"
      unit = "milliseconds"
    }

    detector = {
      auto = {
        tolerance         = "medium"
        training_period   = 86400000
      }
    }
  }
}
`, orgName, projectName, monitorName)
}

func TestAccMonitor_metricAutoDetector(t *testing.T) {
	const (
		orgName     = "acc-mm-auto-org"
		projectName = "acc-mm-auto-project"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccMetricMonitorConfigAuto(orgName, projectName, "acc-mm-auto"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_monitor.test", "id"),
					resource.TestCheckResourceAttr("uptrace_monitor.test", "type", "metric"),
					resource.TestCheckResourceAttr("uptrace_monitor.test", "params_metric.detector.auto.tolerance", "medium"),
					resource.TestCheckResourceAttrSet("uptrace_monitor.test", "status"),
				),
			},
			{
				Config:   testAccMetricMonitorConfigAuto(orgName, projectName, "acc-mm-auto"),
				PlanOnly: true,
			},
			{
				ResourceName:      "uptrace_monitor.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore the whole params_metric block: the backend normalizes MQL
				// (e.g. $http_duration → $http_duration{}), derives column defaults
				// from the query, and populates optional scalars (resolution,
				// num_eval_points, absent_points) — all of which the provider
				// intentionally preserves as null when the user did not set them.
				ImportStateVerifyIgnore: []string{"params_metric"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["uptrace_monitor.test"]
					if !ok {
						return "", fmt.Errorf("uptrace_monitor.test not found in state")
					}
					return fmt.Sprintf("%s:%s", rs.Primary.Attributes["project_id"], rs.Primary.ID), nil
				},
			},
		},
	})
}

func deleteMonitorOutOfBand(t *testing.T, projectIDStr, monitorIDStr string) {
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
		t.Fatalf("delete monitor %d out-of-band: %v", monitorID, err)
	}
}
