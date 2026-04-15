terraform {
  required_providers {
    uptrace = {
      source = "catalin4513/uptrace-ce"
    }
  }
}

provider "uptrace" {
  endpoint   = "http://localhost:14318"
  token      = "user1_secret"
  project_id = 1
}

resource "uptrace_org" "org1" {
  name   = "Org1"
  budget = 100
}

output "org_id" {
  value = uptrace_org.org1.id
}

output "org_name" {
  value = uptrace_org.org1.name
}
