# terraform-provider-uptrace-ce

Minimal Terraform provider for managing Uptrace resources.

## Build

```bash
go build -o terraform-provider-uptrace-ce .
```

## Setup

Create a dev override file so Terraform uses the local binary:

```bash
cat > .terraformrc << 'EOF'
provider_installation {
  dev_overrides {
    "catalin4513/uptrace-ce" = "/path/to/terraform-provider-uptrace-ce"
  }
  direct {}
}
EOF
```

Export it:

```bash
export TF_CLI_CONFIG_FILE=/path/to/terraform-provider-uptrace-ce/.terraformrc
```

## Usage

```hcl
terraform {
  required_providers {
    uptrace = {
      source = "catalin4513/uptrace-ce"
    }
  }
}

provider "uptrace" {
  endpoint   = "http://localhost:14318/internal/v1"
  token      = "user1_secret"
  project_id = 1
}

resource "uptrace_org" "org1" {
  name   = "Org1"
  budget = 100
}
```

Provider config can also be set via environment variables:

- `UPTRACE_ENDPOINT`
- `UPTRACE_TOKEN`
- `UPTRACE_PROJECT_ID`

## Running tests

Integration tests hit a real Uptrace API, so they need credentials. Copy
the committed template to a local `.env` and fill it in:

```bash
cp .env.example .env
```

To keep a created org around for manual inspection in the UI after a
single-subtest run:

```bash
KEEP_TEST_ORG=1 go test ./internal/services/org/... -v -run 'TestOrgResource_CRUD/create_org$'
```

## Commands

```bash
terraform plan      # preview changes
terraform apply     # create/update resources
terraform destroy   # delete resources
terraform state list   # list managed resources
```

## Resources

### uptrace_org

Manages an Uptrace organization.

| Field  | Type   | Required | Note                           |
|--------|--------|----------|--------------------------------|
| name   | string | yes      | Updatable                      |
| budget | float  | no       | Set on creation only (replace) |
| id     | string | computed |                                |

## Files not in git

The `.gitignore` excludes files generated locally:

| File | What it is | How to get it |
|------|-----------|---------------|
| `terraform-provider-uptrace-ce` | Binary | `go build -o terraform-provider-uptrace-ce .` |
| `.terraformrc` / `.tofurc` | Dev override config | Create manually (see Setup) |
| `*.tfstate` | Terraform state | Created by `terraform apply` |
| `.terraform/` | Provider cache | Created by `terraform init` |
| `.terraform.lock.hcl` | Dependency lock | Created by `terraform init` |
| `.env` | Local credentials | `cp .env.example .env` |
