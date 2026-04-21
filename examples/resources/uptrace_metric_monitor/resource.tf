resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

# Metric monitor — anomaly detection on HTTP latency.
resource "uptrace_metric_monitor" "http_latency" {
  project_id = uptrace_project.example.id
  name       = "HTTP latency spike"

  params = {
    metrics = [
      { name = "uptrace_tracing_spans", alias = "$http_duration" }
    ]
    query = "avg($http_duration)"
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
resource "uptrace_metric_monitor" "queue_depth" {
  project_id = uptrace_project.example.id
  name       = "Queue depth threshold"

  params = {
    metrics = [
      { name = "my_queue_depth", alias = "$queue_depth" }
    ]
    query = "sum($queue_depth)"

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
