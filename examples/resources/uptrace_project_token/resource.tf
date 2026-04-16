resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"
}

resource "uptrace_project_token" "example" {
  project_id = uptrace_project.example.id
  name       = "ci-ingest"
}
