resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

resource "uptrace_telegram_channel" "example" {
  project_id = uptrace_project.example.id
  name       = "alerts-telegram"
  priorities = ["high"]
  chat_id    = -100123456
}
