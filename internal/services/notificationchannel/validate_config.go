package notificationchannel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *NotificationChannelResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg notificationChannelModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateMatchAll(&cfg, &resp.Diagnostics)

	if cfg.Type.IsUnknown() {
		return
	}
	typeName := cfg.Type.ValueString()

	validateExactlyOneBlock(&cfg, typeName, &resp.Diagnostics)
	validateSelectedBlock(&cfg, typeName, &resp.Diagnostics)
}

func validateMatchAll(cfg *notificationChannelModel, diags *diag.Diagnostics) {
	// match_all is null by default (true via schema default); only enforce when explicitly false.
	if cfg.MatchAll.IsNull() || cfg.MatchAll.IsUnknown() || cfg.MatchAll.ValueBool() {
		return
	}
	if cfg.MonitorIDs.IsUnknown() {
		return
	}
	if cfg.MonitorIDs.IsNull() || len(cfg.MonitorIDs.Elements()) == 0 {
		diags.AddAttributeError(
			path.Root("monitor_ids"),
			"monitor_ids is required",
			"monitor_ids must be set and non-empty when match_all is false.",
		)
	}
}

func validateExactlyOneBlock(cfg *notificationChannelModel, typeName string, diags *diag.Diagnostics) {
	checks := []struct {
		name  string
		isSet bool
	}{
		{"slack", slackSet(cfg.Slack)},
		{"google_chat", googleChatSet(cfg.GoogleChat)},
		{"mattermost", mattermostSet(cfg.Mattermost)},
		{"pagerduty", pagerdutySet(cfg.Pagerduty)},
		{"servicenow", servicenowSet(cfg.Servicenow)},
		{"opsgenie", opsgenieSet(cfg.Opsgenie)},
		{"telegram", telegramSet(cfg.Telegram)},
		{"teams", teamsSet(cfg.Teams)},
		{"pushover", pushoverSet(cfg.Pushover)},
		{"webhook", webhookSet(cfg.Webhook)},
		{"alertmanager", alertmanagerSet(cfg.Alertmanager)},
		{"incidentio", incidentioSet(cfg.Incidentio)},
	}
	for _, c := range checks {
		switch {
		case c.name == typeName && !c.isSet:
			diags.AddAttributeError(
				path.Root(c.name),
				"missing params block",
				fmt.Sprintf("a %s block is required when type is %q.", c.name, typeName),
			)
		case c.name != typeName && c.isSet:
			diags.AddAttributeError(
				path.Root(c.name),
				"unexpected params block",
				fmt.Sprintf("%s block must not be configured when type is %q.", c.name, typeName),
			)
		}
	}
}

func validateSelectedBlock(cfg *notificationChannelModel, typeName string, diags *diag.Diagnostics) {
	switch typeName {
	case "slack":
		validateSlackBlock(cfg.Slack, diags)
	case "google_chat":
		validateRequiredString(cfg.GoogleChat, "google_chat", "webhook_url", func(m *googleChatModel) types.String { return m.WebhookURL }, diags)
	case "mattermost":
		validateRequiredString(cfg.Mattermost, "mattermost", "webhook_url", func(m *mattermostModel) types.String { return m.WebhookURL }, diags)
	case "pagerduty":
		validatePagerdutyBlock(cfg.Pagerduty, diags)
	case "servicenow":
		validateServicenowBlock(cfg.Servicenow, diags)
	case "opsgenie":
		validateOpsgenieBlock(cfg.Opsgenie, diags)
	case "telegram":
		validateTelegramBlock(cfg.Telegram, diags)
	case "teams":
		validateRequiredString(cfg.Teams, "teams", "webhook_url", func(m *teamsModel) types.String { return m.WebhookURL }, diags)
	case "pushover":
		validatePushoverBlock(cfg.Pushover, diags)
	case "webhook":
		validateRequiredString(cfg.Webhook, "webhook", "url", func(m *webhookModel) types.String { return m.URL }, diags)
	case "alertmanager":
		validateAlertmanagerBlock(cfg.Alertmanager, diags)
	case "incidentio":
		validateIncidentioBlock(cfg.Incidentio, diags)
	}
}

// --- block-set predicates ---

func slackSet(m *slackModel) bool {
	if m == nil {
		return false
	}
	return !m.AuthMethod.IsNull() || !m.WebhookURL.IsNull() || !m.Token.IsNull() || !m.Channel.IsNull()
}

func googleChatSet(m *googleChatModel) bool {
	return m != nil && !m.WebhookURL.IsNull()
}

func mattermostSet(m *mattermostModel) bool {
	return m != nil && !m.WebhookURL.IsNull()
}

func pagerdutySet(m *pagerdutyModel) bool {
	if m == nil {
		return false
	}
	return !m.RoutingKey.IsNull() || !m.Severity.IsNull()
}

func servicenowSet(m *servicenowModel) bool {
	if m == nil {
		return false
	}
	return !m.URL.IsNull() || !m.Username.IsNull() || !m.Password.IsNull() ||
		!m.Category.IsNull() || !m.Subcategory.IsNull() || !m.Impact.IsNull() ||
		!m.Urgency.IsNull() || !m.Severity.IsNull() || !m.CallerID.IsNull() ||
		!m.Group.IsNull() || !m.AssignedTo.IsNull() || !m.OpenedBy.IsNull() ||
		!m.Notify.IsNull() || !m.DueDate.IsNull()
}

func opsgenieSet(m *opsgenieModel) bool {
	if m == nil {
		return false
	}
	return !m.APIKey.IsNull() || !m.Priority.IsNull()
}

func telegramSet(m *telegramModel) bool {
	return m != nil && !m.ChatID.IsNull()
}

func teamsSet(m *teamsModel) bool {
	return m != nil && !m.WebhookURL.IsNull()
}

func pushoverSet(m *pushoverModel) bool {
	if m == nil {
		return false
	}
	return !m.Token.IsNull() || !m.UserKey.IsNull() || !m.Priority.IsNull() || !m.Sound.IsNull()
}

func webhookSet(m *webhookModel) bool {
	return m != nil && !m.URL.IsNull()
}

func alertmanagerSet(m *alertmanagerModel) bool {
	if m == nil {
		return false
	}
	return !m.URL.IsNull() || !m.AuthMethod.IsNull() || !m.Username.IsNull() ||
		!m.Password.IsNull() || !m.Token.IsNull()
}

func incidentioSet(m *incidentioModel) bool {
	if m == nil {
		return false
	}
	return !m.URL.IsNull() || !m.APIKey.IsNull()
}

// --- per-type validators ---

func validateSlackBlock(m *slackModel, diags *diag.Diagnostics) {
	if m == nil {
		return
	}
	if m.AuthMethod.IsUnknown() {
		return
	}
	if m.AuthMethod.IsNull() {
		diags.AddAttributeError(
			path.Root("slack").AtName("auth_method"),
			"auth_method is required",
			`slack.auth_method must be set to "webhook" or "token".`,
		)
		return
	}
	switch m.AuthMethod.ValueString() {
	case "webhook":
		if m.WebhookURL.IsNull() && !m.WebhookURL.IsUnknown() {
			diags.AddAttributeError(
				path.Root("slack").AtName("webhook_url"),
				"webhook_url is required",
				`slack.webhook_url must be set when auth_method is "webhook".`,
			)
		}
		forbidField(m.Token, path.Root("slack").AtName("token"), "auth_method is \"webhook\"", diags)
		forbidField(m.Channel, path.Root("slack").AtName("channel"), "auth_method is \"webhook\"", diags)
	case "token":
		if m.Token.IsNull() && !m.Token.IsUnknown() {
			diags.AddAttributeError(
				path.Root("slack").AtName("token"),
				"token is required",
				`slack.token must be set when auth_method is "token".`,
			)
		}
		if m.Channel.IsNull() && !m.Channel.IsUnknown() {
			diags.AddAttributeError(
				path.Root("slack").AtName("channel"),
				"channel is required",
				`slack.channel must be set when auth_method is "token".`,
			)
		}
		forbidField(m.WebhookURL, path.Root("slack").AtName("webhook_url"), "auth_method is \"token\"", diags)
	}
}

func validatePagerdutyBlock(m *pagerdutyModel, diags *diag.Diagnostics) {
	if m == nil {
		return
	}
	if m.RoutingKey.IsNull() && !m.RoutingKey.IsUnknown() {
		diags.AddAttributeError(
			path.Root("pagerduty").AtName("routing_key"),
			"routing_key is required",
			"pagerduty.routing_key must be set.",
		)
	}
	if m.Severity.IsNull() && !m.Severity.IsUnknown() {
		diags.AddAttributeError(
			path.Root("pagerduty").AtName("severity"),
			"severity is required",
			"pagerduty.severity must be set.",
		)
	}
}

func validateServicenowBlock(m *servicenowModel, diags *diag.Diagnostics) {
	validateRequiredString(m, "servicenow", "url", func(x *servicenowModel) types.String { return x.URL }, diags)
	validateRequiredString(m, "servicenow", "username", func(x *servicenowModel) types.String { return x.Username }, diags)
	validateRequiredString(m, "servicenow", "password", func(x *servicenowModel) types.String { return x.Password }, diags)
}

func validateOpsgenieBlock(m *opsgenieModel, diags *diag.Diagnostics) {
	if m == nil {
		return
	}
	if m.APIKey.IsNull() && !m.APIKey.IsUnknown() {
		diags.AddAttributeError(
			path.Root("opsgenie").AtName("api_key"),
			"api_key is required",
			"opsgenie.api_key must be set.",
		)
	}
	if m.Priority.IsNull() && !m.Priority.IsUnknown() {
		diags.AddAttributeError(
			path.Root("opsgenie").AtName("priority"),
			"priority is required",
			"opsgenie.priority must be set.",
		)
	}
}

func validateTelegramBlock(m *telegramModel, diags *diag.Diagnostics) {
	if m == nil {
		return
	}
	if m.ChatID.IsNull() && !m.ChatID.IsUnknown() {
		diags.AddAttributeError(
			path.Root("telegram").AtName("chat_id"),
			"chat_id is required",
			"telegram.chat_id must be set.",
		)
	}
}

func validatePushoverBlock(m *pushoverModel, diags *diag.Diagnostics) {
	if m == nil {
		return
	}
	if m.Token.IsNull() && !m.Token.IsUnknown() {
		diags.AddAttributeError(
			path.Root("pushover").AtName("token"),
			"token is required",
			"pushover.token must be set.",
		)
	}
	if m.UserKey.IsNull() && !m.UserKey.IsUnknown() {
		diags.AddAttributeError(
			path.Root("pushover").AtName("user_key"),
			"user_key is required",
			"pushover.user_key must be set.",
		)
	}
}

func validateAlertmanagerBlock(m *alertmanagerModel, diags *diag.Diagnostics) {
	if m == nil {
		return
	}
	if m.URL.IsNull() && !m.URL.IsUnknown() {
		diags.AddAttributeError(
			path.Root("alertmanager").AtName("url"),
			"url is required",
			"alertmanager.url must be set.",
		)
	}
	if m.AuthMethod.IsUnknown() || m.AuthMethod.IsNull() {
		return
	}
	switch m.AuthMethod.ValueString() {
	case "basic_auth":
		if m.Username.IsNull() && !m.Username.IsUnknown() {
			diags.AddAttributeError(
				path.Root("alertmanager").AtName("username"),
				"username is required",
				`alertmanager.username must be set when auth_method is "basic_auth".`,
			)
		}
		if m.Password.IsNull() && !m.Password.IsUnknown() {
			diags.AddAttributeError(
				path.Root("alertmanager").AtName("password"),
				"password is required",
				`alertmanager.password must be set when auth_method is "basic_auth".`,
			)
		}
		forbidField(m.Token, path.Root("alertmanager").AtName("token"), `auth_method is "basic_auth"`, diags)
	case "bearer":
		if m.Token.IsNull() && !m.Token.IsUnknown() {
			diags.AddAttributeError(
				path.Root("alertmanager").AtName("token"),
				"token is required",
				`alertmanager.token must be set when auth_method is "bearer".`,
			)
		}
		forbidField(m.Username, path.Root("alertmanager").AtName("username"), `auth_method is "bearer"`, diags)
		forbidField(m.Password, path.Root("alertmanager").AtName("password"), `auth_method is "bearer"`, diags)
	case "none":
		forbidField(m.Username, path.Root("alertmanager").AtName("username"), `auth_method is "none"`, diags)
		forbidField(m.Password, path.Root("alertmanager").AtName("password"), `auth_method is "none"`, diags)
		forbidField(m.Token, path.Root("alertmanager").AtName("token"), `auth_method is "none"`, diags)
	}
}

func validateIncidentioBlock(m *incidentioModel, diags *diag.Diagnostics) {
	if m == nil {
		return
	}
	if m.URL.IsNull() && !m.URL.IsUnknown() {
		diags.AddAttributeError(
			path.Root("incidentio").AtName("url"),
			"url is required",
			"incidentio.url must be set.",
		)
	}
	if m.APIKey.IsNull() && !m.APIKey.IsUnknown() {
		diags.AddAttributeError(
			path.Root("incidentio").AtName("api_key"),
			"api_key is required",
			"incidentio.api_key must be set.",
		)
	}
}

// --- shared helpers ---

type nullable interface {
	IsNull() bool
	IsUnknown() bool
}

func forbidField(v nullable, p path.Path, reason string, diags *diag.Diagnostics) {
	if !v.IsNull() && !v.IsUnknown() {
		diags.AddAttributeError(
			p,
			"unexpected attribute",
			fmt.Sprintf("%s must not be set when %s.", p.String(), reason),
		)
	}
}

func validateRequiredString[T any](m *T, blockName, fieldName string, getter func(*T) types.String, diags *diag.Diagnostics) {
	if m == nil {
		return
	}
	v := getter(m)
	if v.IsNull() && !v.IsUnknown() {
		diags.AddAttributeError(
			path.Root(blockName).AtName(fieldName),
			fieldName+" is required",
			blockName+"."+fieldName+" must be set.",
		)
	}
}
