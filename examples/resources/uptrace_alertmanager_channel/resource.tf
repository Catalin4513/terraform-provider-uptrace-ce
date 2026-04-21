resource "uptrace_alertmanager_channel" "basic_auth" {
  project_id  = uptrace_project.example.id
  name        = "alerts-alertmanager-basic"
  priorities  = ["high"]
  url         = "https://alertmanager.example.com/api/v2/alerts"
  auth_method = "basic_auth"
  username    = "probe"
  password    = "probe-secret"
}

resource "uptrace_alertmanager_channel" "bearer" {
  project_id  = uptrace_project.example.id
  name        = "alerts-alertmanager-bearer"
  priorities  = ["high"]
  url         = "https://alertmanager.example.com/api/v2/alerts"
  auth_method = "bearer"
  token       = "alertmanager-bearer-token"
}
