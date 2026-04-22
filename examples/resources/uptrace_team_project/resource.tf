resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

resource "uptrace_team" "example" {
  org_id = uptrace_org.example.id
  name   = "platform"
}

resource "uptrace_team_project" "example" {
  org_id     = uptrace_org.example.id
  team_id    = uptrace_team.example.id
  project_id = uptrace_project.example.id
}
