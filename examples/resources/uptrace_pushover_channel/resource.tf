resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

resource "uptrace_pushover_channel" "example" {
  project_id = uptrace_project.example.id
  name       = "alerts-pushover"
  priorities = ["high"]
  token      = "your-app-token"
  user_key   = "your-user-key"
  priority   = 1
  sound      = "siren"
}
