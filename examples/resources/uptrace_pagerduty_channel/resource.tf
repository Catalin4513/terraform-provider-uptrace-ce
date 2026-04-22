resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

resource "uptrace_pagerduty_channel" "example" {
  project_id  = uptrace_project.example.id
  name        = "alerts-pagerduty"
  priorities  = ["high"]
  routing_key = "R01ABCDEFGHIJKLMN"
  severity    = "error"
}
