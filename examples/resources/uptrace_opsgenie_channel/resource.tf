resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

resource "uptrace_opsgenie_channel" "example" {
  project_id = uptrace_project.example.id
  name       = "alerts-opsgenie"
  priorities = ["high"]
  api_key    = "your-opsgenie-api-key"
  priority   = "P2"
}
