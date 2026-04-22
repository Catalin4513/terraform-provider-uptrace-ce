resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_team" "example" {
  org_id = uptrace_org.example.id
  name   = "platform"
}

# The org_user_id is the ID of the OrgUser record linking a user to the
# organization (not the User ID). Get it from the Uptrace UI or the
# GET /internal/v1/orgs/{org_id}/teams/{team_id}/users response.
resource "uptrace_team_user" "example" {
  org_id      = uptrace_org.example.id
  team_id     = uptrace_team.example.id
  org_user_id = "42"
}
