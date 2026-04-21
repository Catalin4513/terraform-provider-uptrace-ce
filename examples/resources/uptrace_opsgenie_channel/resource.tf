resource "uptrace_opsgenie_channel" "example" {
  project_id = uptrace_project.example.id
  name       = "alerts-opsgenie"
  priorities = ["high"]
  api_key    = "your-opsgenie-api-key"
  priority   = "P2"
}
