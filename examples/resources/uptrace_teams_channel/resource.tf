resource "uptrace_teams_channel" "example" {
  project_id  = uptrace_project.example.id
  name        = "alerts-teams"
  priorities  = ["high"]
  webhook_url = "https://outlook.office.com/webhook/..."
}
