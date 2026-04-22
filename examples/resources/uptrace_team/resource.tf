resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_team" "example" {
  org_id     = uptrace_org.example.id
  name       = "platform"
  perm_level = "edit"
}
