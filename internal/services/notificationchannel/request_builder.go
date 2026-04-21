package notificationchannel

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

func buildRequestBody(ctx context.Context, m *notificationChannelModel) (*generated.NotificationChannelRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := &generated.NotificationChannelRequest{
		Name: m.Name.ValueString(),
		Type: generated.NotificationChannelRequestType(m.Type.ValueString()),
	}

	// MatchAll.
	if !m.MatchAll.IsNull() && !m.MatchAll.IsUnknown() {
		v := m.MatchAll.ValueBool()
		body.MatchAll = &v
	}

	// Condition.
	if !m.Condition.IsNull() && !m.Condition.IsUnknown() {
		v := m.Condition.ValueString()
		body.Condition = &v
	}

	// Priorities.
	var priorities []types.String
	diags.Append(m.Priorities.ElementsAs(ctx, &priorities, false)...)
	if diags.HasError() {
		return nil, diags
	}
	for _, p := range priorities {
		body.Priorities = append(body.Priorities, generated.NotificationChannelRequestPriorities(p.ValueString()))
	}

	// MonitorIDs.
	if !m.MonitorIDs.IsNull() && !m.MonitorIDs.IsUnknown() {
		var monitorIDs []types.String
		diags.Append(m.MonitorIDs.ElementsAs(ctx, &monitorIDs, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for _, id := range monitorIDs {
			v, err := strconv.ParseInt(id.ValueString(), 10, 64)
			if err != nil {
				diags.AddError("invalid monitor_id", err.Error())
				return nil, diags
			}
			body.MonitorIds = append(body.MonitorIds, v)
		}
	}

	// Params — set the oneOf based on channel type.
	oneOf := &generated.NotificationChannelRequest_Params_OneOf{}

	switch m.Type.ValueString() {
	case "slack":
		if m.Slack == nil {
			diags.AddError("missing slack block", "slack block is required when type is slack")
			return nil, diags
		}
		p := generated.SlackParams{}
		if !m.Slack.AuthMethod.IsNull() && !m.Slack.AuthMethod.IsUnknown() {
			v := generated.SlackParamsAuthMethod(m.Slack.AuthMethod.ValueString())
			p.AuthMethod = &v
		}
		if !m.Slack.WebhookURL.IsNull() && !m.Slack.WebhookURL.IsUnknown() {
			v := m.Slack.WebhookURL.ValueString()
			p.WebhookURL = &v
		}
		if !m.Slack.Token.IsNull() && !m.Slack.Token.IsUnknown() {
			v := m.Slack.Token.ValueString()
			p.Token = &v
		}
		if !m.Slack.Channel.IsNull() && !m.Slack.Channel.IsUnknown() {
			v := m.Slack.Channel.ValueString()
			p.Channel = &v
		}
		if err := oneOf.FromSlackParams(p); err != nil {
			diags.AddError("failed to set slack params", err.Error())
			return nil, diags
		}

	case "google_chat":
		if m.GoogleChat == nil {
			diags.AddError("missing google_chat block", "google_chat block is required when type is google_chat")
			return nil, diags
		}
		if err := oneOf.FromGoogleChatParams(generated.GoogleChatParams{
			WebhookURL: m.GoogleChat.WebhookURL.ValueString(),
		}); err != nil {
			diags.AddError("failed to set google_chat params", err.Error())
			return nil, diags
		}

	case "mattermost":
		if m.Mattermost == nil {
			diags.AddError("missing mattermost block", "mattermost block is required when type is mattermost")
			return nil, diags
		}
		if err := oneOf.FromMattermostParams(generated.MattermostParams{
			WebhookURL: m.Mattermost.WebhookURL.ValueString(),
		}); err != nil {
			diags.AddError("failed to set mattermost params", err.Error())
			return nil, diags
		}

	case "pagerduty":
		if m.Pagerduty == nil {
			diags.AddError("missing pagerduty block", "pagerduty block is required when type is pagerduty")
			return nil, diags
		}
		if err := oneOf.FromPagerdutyParams(generated.PagerdutyParams{
			RoutingKey: m.Pagerduty.RoutingKey.ValueString(),
			Severity:   generated.PagerdutyParamsSeverity(m.Pagerduty.Severity.ValueString()),
		}); err != nil {
			diags.AddError("failed to set pagerduty params", err.Error())
			return nil, diags
		}

	case "servicenow":
		if m.Servicenow == nil {
			diags.AddError("missing servicenow block", "servicenow block is required when type is servicenow")
			return nil, diags
		}
		p := generated.ServicenowParams{
			URL:      m.Servicenow.URL.ValueString(),
			Username: m.Servicenow.Username.ValueString(),
			Password: m.Servicenow.Password.ValueString(),
		}
		p.Category = valueStringToPtr(m.Servicenow.Category)
		p.Subcategory = valueStringToPtr(m.Servicenow.Subcategory)
		if !m.Servicenow.Impact.IsNull() && !m.Servicenow.Impact.IsUnknown() {
			v := generated.ServicenowParamsImpact(m.Servicenow.Impact.ValueString())
			p.Impact = &v
		}
		if !m.Servicenow.Urgency.IsNull() && !m.Servicenow.Urgency.IsUnknown() {
			v := generated.ServicenowParamsUrgency(m.Servicenow.Urgency.ValueString())
			p.Urgency = &v
		}
		if !m.Servicenow.Severity.IsNull() && !m.Servicenow.Severity.IsUnknown() {
			v := generated.ServicenowParamsSeverity(m.Servicenow.Severity.ValueString())
			p.Severity = &v
		}
		p.CallerID = valueStringToPtr(m.Servicenow.CallerID)
		p.Group = valueStringToPtr(m.Servicenow.Group)
		p.AssignedTo = valueStringToPtr(m.Servicenow.AssignedTo)
		p.OpenedBy = valueStringToPtr(m.Servicenow.OpenedBy)
		if !m.Servicenow.Notify.IsNull() && !m.Servicenow.Notify.IsUnknown() {
			v := generated.ServicenowParamsNotify(m.Servicenow.Notify.ValueString())
			p.Notify = &v
		}
		p.DueDate = valueStringToPtr(m.Servicenow.DueDate)
		if err := oneOf.FromServicenowParams(p); err != nil {
			diags.AddError("failed to set servicenow params", err.Error())
			return nil, diags
		}

	case "opsgenie":
		if m.Opsgenie == nil {
			diags.AddError("missing opsgenie block", "opsgenie block is required when type is opsgenie")
			return nil, diags
		}
		if err := oneOf.FromOpsgenieParams(generated.OpsgenieParams{
			APIKey:   m.Opsgenie.APIKey.ValueString(),
			Priority: generated.OpsgenieParamsPriority(m.Opsgenie.Priority.ValueString()),
		}); err != nil {
			diags.AddError("failed to set opsgenie params", err.Error())
			return nil, diags
		}

	case "telegram":
		if m.Telegram == nil {
			diags.AddError("missing telegram block", "telegram block is required when type is telegram")
			return nil, diags
		}
		if err := oneOf.FromTelegramParams(generated.TelegramParams{
			ChatID: m.Telegram.ChatID.ValueInt64(),
		}); err != nil {
			diags.AddError("failed to set telegram params", err.Error())
			return nil, diags
		}

	case "teams":
		if m.Teams == nil {
			diags.AddError("missing teams block", "teams block is required when type is teams")
			return nil, diags
		}
		if err := oneOf.FromTeamsParams(generated.TeamsParams{
			WebhookURL: m.Teams.WebhookURL.ValueString(),
		}); err != nil {
			diags.AddError("failed to set teams params", err.Error())
			return nil, diags
		}

	case "pushover":
		if m.Pushover == nil {
			diags.AddError("missing pushover block", "pushover block is required when type is pushover")
			return nil, diags
		}
		p := generated.PushoverParams{
			Token:   m.Pushover.Token.ValueString(),
			UserKey: m.Pushover.UserKey.ValueString(),
		}
		if !m.Pushover.Priority.IsNull() && !m.Pushover.Priority.IsUnknown() {
			v := int(m.Pushover.Priority.ValueInt64())
			p.Priority = &v
		}
		if !m.Pushover.Sound.IsNull() && !m.Pushover.Sound.IsUnknown() {
			v := m.Pushover.Sound.ValueString()
			p.Sound = &v
		}
		if err := oneOf.FromPushoverParams(p); err != nil {
			diags.AddError("failed to set pushover params", err.Error())
			return nil, diags
		}

	case "webhook":
		if m.Webhook == nil {
			diags.AddError("missing webhook block", "webhook block is required when type is webhook")
			return nil, diags
		}
		rawParams := map[string]any{
			"url": m.Webhook.URL.ValueString(),
		}
		if !m.Webhook.Payload.IsNull() && !m.Webhook.Payload.IsUnknown() {
			var payload map[string]any
			if err := json.Unmarshal([]byte(m.Webhook.Payload.ValueString()), &payload); err != nil {
				diags.AddError("invalid webhook payload", err.Error())
				return nil, diags
			}
			rawParams["payload"] = payload
		}
		rawJSON, err := json.Marshal(rawParams)
		if err != nil {
			diags.AddError("failed to encode webhook params", err.Error())
			return nil, diags
		}
		if err := oneOf.UnmarshalJSON(rawJSON); err != nil {
			diags.AddError("failed to set webhook params", err.Error())
			return nil, diags
		}

	case "alertmanager":
		if m.Alertmanager == nil {
			diags.AddError("missing alertmanager block", "alertmanager block is required when type is alertmanager")
			return nil, diags
		}
		p := generated.AlertmanagerParams{
			URL: m.Alertmanager.URL.ValueString(),
		}
		if !m.Alertmanager.AuthMethod.IsNull() && !m.Alertmanager.AuthMethod.IsUnknown() {
			v := generated.AlertmanagerParamsAuthMethod(m.Alertmanager.AuthMethod.ValueString())
			p.AuthMethod = &v
		}
		p.Username = valueStringToPtr(m.Alertmanager.Username)
		p.Password = valueStringToPtr(m.Alertmanager.Password)
		p.Token = valueStringToPtr(m.Alertmanager.Token)
		if err := oneOf.FromAlertmanagerParams(p); err != nil {
			diags.AddError("failed to set alertmanager params", err.Error())
			return nil, diags
		}

	case "incidentio":
		if m.Incidentio == nil {
			diags.AddError("missing incidentio block", "incidentio block is required when type is incidentio")
			return nil, diags
		}
		if err := oneOf.FromIncidentioParams(generated.IncidentioParams{
			URL:    m.Incidentio.URL.ValueString(),
			APIKey: m.Incidentio.APIKey.ValueString(),
		}); err != nil {
			diags.AddError("failed to set incidentio params", err.Error())
			return nil, diags
		}

	default:
		diags.AddError("unsupported channel type", "type "+m.Type.ValueString()+" is not supported")
		return nil, diags
	}

	body.Params = generated.NotificationChannelRequest_Params{
		NotificationChannelRequest_Params_OneOf: oneOf,
	}

	return body, diags
}

func valueStringToPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}
