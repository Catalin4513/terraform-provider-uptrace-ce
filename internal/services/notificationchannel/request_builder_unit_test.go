package notificationchannel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestBody_slackWebhook(t *testing.T) {
	priorities, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"high"})
	m := &notificationChannelModel{
		Name:       types.StringValue("slack-alerts"),
		Type:       types.StringValue("slack"),
		MatchAll:   types.BoolValue(true),
		Condition:  types.StringNull(),
		Priorities: priorities,
		MonitorIDs: types.SetNull(types.StringType),
		Slack: &slackModel{
			AuthMethod: types.StringValue("webhook"),
			WebhookURL: types.StringValue("https://hooks.slack.com/services/T00/B00/XXXX"),
			Token:      types.StringNull(),
			Channel:    types.StringNull(),
		},
	}

	body, diags := buildRequestBody(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, "slack", string(body.Type))

	p, err := body.Params.NotificationChannelRequest_Params_OneOf.AsSlackParams()
	require.NoError(t, err)
	require.NotNil(t, p.AuthMethod)
	require.Equal(t, "webhook", string(*p.AuthMethod))
	require.NotNil(t, p.WebhookURL)
	require.Equal(t, "https://hooks.slack.com/services/T00/B00/XXXX", *p.WebhookURL)
	require.Nil(t, p.Token)
	require.Nil(t, p.Channel)
}

func TestBuildRequestBody_slackToken(t *testing.T) {
	priorities, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"high", "medium"})
	m := &notificationChannelModel{
		Name:       types.StringValue("slack-bot"),
		Type:       types.StringValue("slack"),
		MatchAll:   types.BoolValue(true),
		Condition:  types.StringNull(),
		Priorities: priorities,
		MonitorIDs: types.SetNull(types.StringType),
		Slack: &slackModel{
			AuthMethod: types.StringValue("token"),
			WebhookURL: types.StringNull(),
			Token:      types.StringValue("xoxb-secret"),
			Channel:    types.StringValue("#alerts"),
		},
	}

	body, diags := buildRequestBody(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)

	p, err := body.Params.NotificationChannelRequest_Params_OneOf.AsSlackParams()
	require.NoError(t, err)
	require.NotNil(t, p.AuthMethod)
	require.Equal(t, "token", string(*p.AuthMethod))
	require.Nil(t, p.WebhookURL)
	require.NotNil(t, p.Token)
	require.Equal(t, "xoxb-secret", *p.Token)
	require.NotNil(t, p.Channel)
	require.Equal(t, "#alerts", *p.Channel)
}

func TestBuildRequestBody_webhook(t *testing.T) {
	priorities, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"high", "low"})
	m := &notificationChannelModel{
		Name:       types.StringValue("alerts-hook"),
		Type:       types.StringValue("webhook"),
		MatchAll:   types.BoolValue(true),
		Condition:  types.StringNull(),
		Priorities: priorities,
		MonitorIDs: types.SetNull(types.StringType),
		Webhook: &webhookModel{
			URL: types.StringValue("https://example.com/hook"),
		},
	}

	body, diags := buildRequestBody(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.Equal(t, "alerts-hook", body.Name)
	require.Equal(t, "webhook", string(body.Type))
	require.NotNil(t, body.MatchAll)
	require.True(t, *body.MatchAll)
	require.Nil(t, body.Condition)
	require.Len(t, body.Priorities, 2)
	require.Equal(t, "high", string(body.Priorities[0]))
	require.Equal(t, "low", string(body.Priorities[1]))
	require.Nil(t, body.MonitorIds)

	p, err := body.Params.NotificationChannelRequest_Params_OneOf.AsWebhookParams()
	require.NoError(t, err)
	require.Equal(t, "https://example.com/hook", p.URL)
}

func TestBuildRequestBody_webhookWithPayload(t *testing.T) {
	priorities, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"high"})
	m := &notificationChannelModel{
		Name:       types.StringValue("alerts-hook"),
		Type:       types.StringValue("webhook"),
		MatchAll:   types.BoolValue(true),
		Condition:  types.StringNull(),
		Priorities: priorities,
		MonitorIDs: types.SetNull(types.StringType),
		Webhook: &webhookModel{
			URL:     types.StringValue("https://example.com/hook"),
			Payload: types.StringValue(`{"alert":"uptrace","severity":"high","tags":["prod","api"]}`),
		},
	}

	body, diags := buildRequestBody(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)

	p, err := body.Params.NotificationChannelRequest_Params_OneOf.AsWebhookParams()
	require.NoError(t, err)
	require.Equal(t, "uptrace", p.Payload["alert"])
	require.Equal(t, "high", p.Payload["severity"])
	require.ElementsMatch(t, []any{"prod", "api"}, p.Payload["tags"])
}

func TestBuildRequestBody_webhookWithEmptyPayload(t *testing.T) {
	priorities, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"high"})
	m := &notificationChannelModel{
		Name:       types.StringValue("alerts-hook"),
		Type:       types.StringValue("webhook"),
		MatchAll:   types.BoolValue(true),
		Condition:  types.StringNull(),
		Priorities: priorities,
		MonitorIDs: types.SetNull(types.StringType),
		Webhook: &webhookModel{
			URL:     types.StringValue("https://example.com/hook"),
			Payload: types.StringValue(`{}`),
		},
	}

	body, diags := buildRequestBody(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)

	p, err := body.Params.NotificationChannelRequest_Params_OneOf.AsWebhookParams()
	require.NoError(t, err)
	require.NotNil(t, p.Payload)
	require.Len(t, p.Payload, 0)
}

func TestBuildRequestBody_webhookInvalidPayload(t *testing.T) {
	priorities, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"high"})
	m := &notificationChannelModel{
		Name:       types.StringValue("alerts-hook"),
		Type:       types.StringValue("webhook"),
		MatchAll:   types.BoolValue(true),
		Condition:  types.StringNull(),
		Priorities: priorities,
		MonitorIDs: types.SetNull(types.StringType),
		Webhook: &webhookModel{
			URL:     types.StringValue("https://example.com/hook"),
			Payload: types.StringValue(`{not valid json`),
		},
	}

	body, diags := buildRequestBody(context.Background(), m)
	require.True(t, diags.HasError())
	require.Nil(t, body)
	require.Contains(t, diags.Errors()[0].Summary(), "invalid webhook payload")
}

func TestBuildRequestBody_withMonitorIDs(t *testing.T) {
	priorities, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"high"})
	monitorIDs, _ := types.SetValueFrom(context.Background(), types.StringType, []string{"10", "20", "30"})
	m := &notificationChannelModel{
		Name:       types.StringValue("test"),
		Type:       types.StringValue("telegram"),
		MatchAll:   types.BoolValue(false),
		Condition:  types.StringNull(),
		Priorities: priorities,
		MonitorIDs: monitorIDs,
		Telegram: &telegramModel{
			ChatID: types.Int64Value(-100123),
		},
	}

	body, diags := buildRequestBody(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.ElementsMatch(t, []int64{10, 20, 30}, body.MonitorIds)
}

func TestBuildRequestBody_invalidMonitorID(t *testing.T) {
	priorities, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"high"})
	monitorIDs, _ := types.SetValueFrom(context.Background(), types.StringType, []string{"abc"})
	m := &notificationChannelModel{
		Name:       types.StringValue("test"),
		Type:       types.StringValue("telegram"),
		MatchAll:   types.BoolValue(false),
		Condition:  types.StringNull(),
		Priorities: priorities,
		MonitorIDs: monitorIDs,
		Telegram: &telegramModel{
			ChatID: types.Int64Value(-100123),
		},
	}

	body, diags := buildRequestBody(context.Background(), m)
	require.True(t, diags.HasError())
	require.Nil(t, body)
}

func TestBuildRequestBody_missingParamBlock(t *testing.T) {
	priorities, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"high"})
	m := &notificationChannelModel{
		Name:       types.StringValue("test"),
		Type:       types.StringValue("slack"),
		MatchAll:   types.BoolValue(true),
		Condition:  types.StringNull(),
		Priorities: priorities,
		MonitorIDs: types.SetNull(types.StringType),
		// Slack block intentionally nil.
	}

	body, diags := buildRequestBody(context.Background(), m)
	require.True(t, diags.HasError())
	require.Nil(t, body)
	require.Contains(t, diags.Errors()[0].Detail(), "slack block is required")
}

func TestBuildRequestBody_unsupportedType(t *testing.T) {
	priorities, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"high"})
	m := &notificationChannelModel{
		Name:       types.StringValue("test"),
		Type:       types.StringValue("carrier_pigeon"),
		MatchAll:   types.BoolValue(true),
		Condition:  types.StringNull(),
		Priorities: priorities,
		MonitorIDs: types.SetNull(types.StringType),
	}

	body, diags := buildRequestBody(context.Background(), m)
	require.True(t, diags.HasError())
	require.Nil(t, body)
	require.Contains(t, diags.Errors()[0].Detail(), "carrier_pigeon")
}

func TestBuildRequestBody_conditionSet(t *testing.T) {
	priorities, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"high"})
	m := &notificationChannelModel{
		Name:       types.StringValue("test"),
		Type:       types.StringValue("webhook"),
		MatchAll:   types.BoolValue(true),
		Condition:  types.StringValue("severity >= warning"),
		Priorities: priorities,
		MonitorIDs: types.SetNull(types.StringType),
		Webhook: &webhookModel{
			URL: types.StringValue("https://example.com"),
		},
	}

	body, diags := buildRequestBody(context.Background(), m)
	require.False(t, diags.HasError(), "unexpected errors: %v", diags)
	require.NotNil(t, body.Condition)
	require.Equal(t, "severity >= warning", *body.Condition)
}

func TestValueStringToPtr_value(t *testing.T) {
	p := valueStringToPtr(types.StringValue("hello"))
	require.NotNil(t, p)
	require.Equal(t, "hello", *p)
}

func TestValueStringToPtr_null(t *testing.T) {
	p := valueStringToPtr(types.StringNull())
	require.Nil(t, p)
}

func TestValueStringToPtr_unknown(t *testing.T) {
	p := valueStringToPtr(types.StringUnknown())
	require.Nil(t, p)
}
