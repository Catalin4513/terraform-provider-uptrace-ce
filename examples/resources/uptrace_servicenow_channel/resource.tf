resource "uptrace_servicenow_channel" "example" {
  project_id = uptrace_project.example.id
  name       = "alerts-servicenow"
  priorities = ["high"]

  url      = "https://instance.service-now.com"
  username = "alertuser"
  password = "alertsecret"

  category = "software"
  impact   = "2"
  urgency  = "1"
  severity = "2"
}
