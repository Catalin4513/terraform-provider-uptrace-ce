resource "uptrace_org" "example" {
  name = "My Organization"
}

resource "uptrace_project" "example" {
  org_id = uptrace_org.example.id
  name   = "api"

  # Durations are milliseconds. Omit to use the server default.
  span_retention = 720 * 60 * 60 * 1000 # 720h
}
