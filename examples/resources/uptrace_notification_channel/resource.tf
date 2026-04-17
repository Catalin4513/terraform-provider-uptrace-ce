resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

resource "uptrace_notification_channel" "slack" {
  project_id = uptrace_project.example.id
  name       = "alerts-slack"
  type       = "slack"
  priorities = ["high", "medium"]
  match_all  = true

  slack {
    auth_method = "webhook"
    webhook_url = "https://hooks.slack.com/services/T00/B00/XXXX"
  }
}

resource "uptrace_notification_channel" "telegram" {
  project_id = uptrace_project.example.id
  name       = "alerts-telegram"
  type       = "telegram"
  priorities = ["high"]
  match_all  = true

  telegram {
    chat_id = -100123456
  }
}

resource "uptrace_notification_channel" "webhook" {
  project_id = uptrace_project.example.id
  name       = "alerts-webhook"
  type       = "webhook"
  priorities = ["high"]

  webhook {
    url = "https://example.com/alerts"
    payload = jsonencode({
      source   = "uptrace"
      severity = "high"
    })
  }
}
