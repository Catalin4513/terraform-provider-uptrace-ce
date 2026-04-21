resource "uptrace_slack_channel" "webhook" {
  project_id = uptrace_project.example.id
  name       = "alerts-slack-webhook"
  priorities = ["high", "medium"]

  auth_method = "webhook"
  webhook_url = "https://hooks.slack.com/services/T00/B00/XXXX"
}

resource "uptrace_slack_channel" "bot" {
  project_id = uptrace_project.example.id
  name       = "alerts-slack-bot"
  priorities = ["high"]

  auth_method = "token"
  token       = "xoxb-your-bot-token"
  channel     = "#alerts"
}
