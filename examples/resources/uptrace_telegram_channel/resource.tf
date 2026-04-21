resource "uptrace_telegram_channel" "example" {
  project_id = uptrace_project.example.id
  name       = "alerts-telegram"
  priorities = ["high"]
  chat_id    = -100123456
}
