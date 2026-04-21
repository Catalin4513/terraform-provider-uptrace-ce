resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

# Error monitor — fires when error trends anomalously.
resource "uptrace_error_monitor" "log_errors" {
  project_id = uptrace_project.example.id
  name       = "Notify on all errors"

  trend_agg_func    = "sum"
  trend_sensitivity = "medium"

  params = {
    metrics = [
      { name = "uptrace_tracing_logs", alias = "$logs" }
    ]
    query = "sum($logs) | where _system in (\"log:error\", \"log:fatal\")"
  }
}
