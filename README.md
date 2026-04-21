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

### uptrace_project_token

Manages an ingest token for an Uptrace project.

| Field      | Type   | Required | Note                                                              |
|------------|--------|----------|-------------------------------------------------------------------|
| project_id | string | yes      | Forces replacement on change.                                     |
| name       | string | no       | Updatable. Removing the attribute clears the name on the server.  |
| id         | string | computed |                                                                   |
| token      | string | computed | Sensitive. Generated server-side; never supplied by the user.     |
| dsn        | string | computed | Sensitive. Ingest URL with the token embedded.                    |

Import with `<project_id>:<token_id>`.

### uptrace_notification_channel

Manages an Uptrace notification channel. Exactly one param block must be configured, and it must match `type` — the provider rejects missing or extra blocks at plan time.

| Field       | Type         | Required | Note                                                                                |
|-------------|--------------|----------|-------------------------------------------------------------------------------------|
| project_id  | string       | yes      | Forces replacement on change.                                                        |
| name        | string       | yes      | Updatable.                                                                           |
| type        | string       | yes      | Forces replacement. One of: `slack`, `google_chat`, `mattermost`, `pagerduty`, `servicenow`, `opsgenie`, `telegram`, `teams`, `pushover`, `webhook`, `alertmanager`, `incidentio`. |
| priorities  | list(string) | yes      | Alert priorities to match. Each value one of `info`, `low`, `medium`, `high`.        |
| match_all   | bool         | no       | Defaults to `true`. When `false`, `monitor_ids` must be set and non-empty.           |
| monitor_ids | list(string) | no       | Required when `match_all = false`.                                                   |
| condition   | string       | no       | Alert condition expression.                                                          |
| id          | string       | computed |                                                                                      |
| status      | string       | computed | One of `delivering`, `paused`, `disabled`, `draft`.                                  |

Type-specific params go in a single nested block named for the `type`:

| Block          | Fields                                                                                                               |
|----------------|----------------------------------------------------------------------------------------------------------------------|
| `slack`        | `auth_method` (`webhook` or `token`), `webhook_url`, `token`, `channel`. Fields required depend on `auth_method`.    |
| `google_chat`  | `webhook_url`                                                                                                        |
| `mattermost`   | `webhook_url`                                                                                                        |
| `pagerduty`    | `routing_key`, `severity` (`critical`, `error`, `warning`, `info`)                                                   |
| `servicenow`   | `url`, `username`, `password`, plus optional `category`, `subcategory`, `impact`, `urgency`, `severity`, `caller_id`, `group`, `assigned_to`, `opened_by`, `notify`, `due_date` |
| `opsgenie`     | `api_key`, `priority`                                                                                                |
| `telegram`     | `chat_id` (int64)                                                                                                    |
| `teams`        | `webhook_url`                                                                                                        |
| `pushover`     | `token`, `user_key`, optional `priority` (int, -2 to 2), `sound`                                                     |
| `webhook`      | `url`, optional `payload` (JSON object string — use `jsonencode()`)                                                  |
| `alertmanager` | `url`, optional `auth_method` (`none`, `basic_auth`, `bearer`), `username`, `password`, `token`. Credential fields required depend on `auth_method`. |
| `incidentio`   | `url`, `api_key`                                                                                                     |

### uptrace_monitor

Manages an Uptrace monitor. Set `type` to `error` or `metric` and configure the matching params block.

Shared fields (both types):

| Field                     | Type              | Required | Note                                                                                                |
|---------------------------|-------------------|----------|-----------------------------------------------------------------------------------------------------|
| project_id                | string            | yes      | Forces replacement on change.                                                                       |
| name                      | string            | yes      | Updatable.                                                                                          |
| type                      | string            | yes      | One of `error`, `metric`. Forces replacement on change.                                             |
| notify_everyone_by_email  | bool              | no       | Defaults to `false`. Updatable.                                                                     |
| trend_agg_func            | string            | no       | Defaults to `sum`. One of `sum`, `avg`, `median`, `last`.                                           |
| trend_sensitivity         | string            | no       | Defaults to `medium`. One of `low`, `medium`, `high`.                                               |
| team_ids                  | set(string)       | no       | Team IDs to notify when the monitor fires.                                                          |
| channel_ids               | set(string)       | no       | Notification channel IDs (`uptrace_notification_channel.id`). No `tonumber()` wrapper needed.       |
| id                        | string            | computed |                                                                                                     |
| status                    | string            | computed | One of `active`, `paused`, `firing`, `no_data`, `disabled`.                                         |

Exactly one of `params_error` / `params_metric` must be set, matching `type`.

`params_error` (required when `type = "error"`):

| Field                     | Type              | Required | Note                                                                                                |
|---------------------------|-------------------|----------|-----------------------------------------------------------------------------------------------------|
| query                     | string            | yes      | MQL query expression. The backend normalizes MQL; the provider preserves the user's input form.     |
| metrics                   | list of objects   | yes      | At least one metric. Each: `{ name = "...", alias = "$..." }`. Aliases must start with `$`.         |

`params_metric` (required when `type = "metric"`):

| Field           | Type            | Required | Note                                                                                               |
|-----------------|-----------------|----------|----------------------------------------------------------------------------------------------------|
| query           | string          | yes      | MQL query expression.                                                                              |
| metrics         | list of objects | yes      | At least one metric. Each: `{ name = "...", alias = "$..." }`.                                     |
| column          | object          | no       | `{ name = "...", unit = "milliseconds" }`. The result column the detector evaluates.               |
| resolution      | number          | no       | Evaluation resolution in milliseconds.                                                             |
| num_eval_points | number          | no       | Number of consecutive evaluation points that must breach the threshold.                            |
| absent_points   | string          | no       | One of `ignore`, `alert`, `zero`.                                                                  |
| time_offset     | number          | no       | Time offset in milliseconds applied to the query before evaluation.                                |
| detector        | object          | yes      | Exactly one of `manual {}` or `auto {}`.                                                           |

`params_metric.detector.manual`:

| Field      | Type   | Required | Note                                                                 |
|------------|--------|----------|----------------------------------------------------------------------|
| min_value  | number | no       | Alert when value falls below this threshold.                         |
| max_value  | number | no       | Alert when value rises above this threshold.                         |
| recovery   | object | no       | Hysteresis `{ min_value, max_value }` used to clear an active alert. |

`params_metric.detector.auto`:

| Field            | Type   | Required | Note                                               |
|------------------|--------|----------|----------------------------------------------------|
| tolerance        | string | no       | One of `low`, `medium`, `high`.                    |
| training_period  | number | no       | Training period in milliseconds.                   |
| min_dev_fraction | number | no       | Minimum deviation as a fraction of the baseline.   |
| min_dev_absolute | number | no       | Minimum absolute deviation from the baseline.      |

Not yet exposed: `repeat_interval` (shared oneOf of `default` / `fixed` / `linear` / `exponential`). Follow-up work.



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
