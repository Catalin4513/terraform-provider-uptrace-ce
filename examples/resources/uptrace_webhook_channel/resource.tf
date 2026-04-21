resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

resource "uptrace_webhook_channel" "example" {
  project_id = uptrace_project.example.id
  name       = "alerts-webhook"
  priorities = ["high"]
  url        = "https://example.com/hooks/alert"

  payload = jsonencode({
    source   = "uptrace"
    severity = "high"
  })
}
