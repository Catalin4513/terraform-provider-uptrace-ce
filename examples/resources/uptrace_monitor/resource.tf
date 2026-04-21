resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

# Error monitor — fires when error trends anomalously.
resource "uptrace_monitor" "log_errors" {
  project_id = uptrace_project.example.id
  name       = "Notify on all errors"
  type       = "error"

  trend_agg_func    = "sum"
  trend_sensitivity = "medium"

  params_error = {
    query = "sum($logs) | where _system in (\"log:error\", \"log:fatal\")"
    metrics = [
      { name = "uptrace_tracing_logs", alias = "$logs" }
    ]
  }
}

# Metric monitor — anomaly detection on HTTP latency.
resource "uptrace_monitor" "http_latency" {
  project_id = uptrace_project.example.id
  name       = "HTTP latency spike"
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
        tolerance       = "medium"
        training_period = 86400000 # 24h in ms
      }
    }
  }
}

# Metric monitor — manual threshold.
resource "uptrace_monitor" "queue_depth" {
  project_id = uptrace_project.example.id
  name       = "Queue depth threshold"
  type       = "metric"

  params_metric = {
    query = "sum($queue_depth)"
    metrics = [
      { name = "my_queue_depth", alias = "$queue_depth" }
    ]

    detector = {
      manual = {
        max_value = 1000
        recovery = {
          max_value = 800
        }
      }
    }
  }
}
