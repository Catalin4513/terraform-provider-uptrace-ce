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
  endpoint = "http://localhost:14318"
  token    = "user1_secret"
}

resource "uptrace_org" "org1" {
  name   = "Org1"
  budget = 100
}
```

Provider config can also be set via environment variables:

- `UPTRACE_ENDPOINT`
- `UPTRACE_TOKEN`

## Testing

The project has two layers of tests:

### Unit tests

Unit tests run without any external dependencies and are always safe to run:

```bash
make test
```

These test pure helper functions (`orgToModel`, `parseOrgID`) and run in CI
on every push and pull request.

### Acceptance tests

Acceptance tests exercise the full Terraform lifecycle (plan, apply, import,
destroy) against a real Uptrace API. They follow the
[HashiCorp acceptance test conventions](https://developer.hashicorp.com/terraform/plugin/testing/acceptance-tests)
and are gated behind the `TF_ACC` environment variable.

1. Copy the credentials template and fill it in:

```bash
cp .env.example .env
```

2. Run acceptance tests:

```bash
make testacc
```

Without `TF_ACC=1`, acceptance tests are automatically skipped.

### Writing tests

- **Unit tests** go in `*_unit_test.go` files with `package <name>` (internal).
  Use these for pure functions that don't need a running API.
- **Acceptance tests** go in `*_acc_test.go` files with `package <name>_test` (external).
  Use the `TestAcc` prefix and `resource.TestCase` with `testutil.ProtoV6ProviderFactories`.
  Always include `PreCheck`, `CheckDestroy`, and an import step.

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
| budget | float  | no       | Updatable. Uses the API default when omitted on create. Removing it later keeps the current API budget because Uptrace does not expose an unset/reset operation. |
| id     | string | computed |                                |

### uptrace_project

Manages an Uptrace project scoped under an organization.

| Field                  | Type   | Required | Note                                                                        |
|------------------------|--------|----------|-----------------------------------------------------------------------------|
| org_id                 | string | yes      | Forces replacement on change.                                               |
| name                   | string | yes      | Updatable.                                                                  |
| group_by_env           | bool   | no       | Updatable.                                                                  |
| group_funcs_by_service | bool   | no       | Updatable.                                                                  |
| semconv_version        | string | no       | One of `none`, `v1.25.0`, `v1.33.0`.                                        |
| display_log_severity   | bool   | no       | Updatable.                                                                  |
| count_distinct         | bool   | no       | Updatable.                                                                  |
| span_time_range        | string | no       | Duration (e.g. `"24h"`). Default query time range for spans.                |
| log_time_range         | string | no       | Duration. Default query time range for logs.                                |
| event_time_range       | string | no       | Duration. Default query time range for events.                              |
| span_retention         | string | no       | Duration (e.g. `"30d"`, `"4w"`). Server minimum applies.                    |
| log_retention          | string | no       | Duration.                                                                   |
| event_retention        | string | no       | Duration.                                                                   |
| metric_retention       | string | no       | Duration.                                                                   |
| id                     | string | computed |                                                                             |

Duration strings accept the stdlib units `ns`, `us`, `ms`, `s`, `m`, `h` plus `d` (day) and `w` (week).


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
