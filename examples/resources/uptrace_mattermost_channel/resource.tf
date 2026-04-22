resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

resource "uptrace_mattermost_channel" "example" {
  project_id  = uptrace_project.example.id
  name        = "alerts-mattermost"
  priorities  = ["high"]
  webhook_url = "https://mattermost.example.com/hooks/abcdef"
}
