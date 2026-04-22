resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

resource "uptrace_google_chat_channel" "example" {
  project_id  = uptrace_project.example.id
  name        = "alerts-google-chat"
  priorities  = ["high"]
  webhook_url = "https://chat.googleapis.com/v1/spaces/AAAA/messages?key=K&token=T"
}
