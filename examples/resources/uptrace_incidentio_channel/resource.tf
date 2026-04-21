resource "uptrace_incidentio_channel" "example" {
  project_id = uptrace_project.example.id
  name       = "alerts-incidentio"
  priorities = ["high"]
  url        = "https://api.incident.io/v2/alert_events/..."
  api_key    = "your-incidentio-api-key"
}
