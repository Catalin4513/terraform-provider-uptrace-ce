package notificationchannel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

func newOneOfWithSlack(p generated.SlackParams) *generated.NotificationChannel_Params_OneOf {
	o := &generated.NotificationChannel_Params_OneOf{}
	_ = o.FromSlackParams(p)
	return o
}

func newOneOfWithTelegram(p generated.TelegramParams) *generated.NotificationChannel_Params_OneOf {
	o := &generated.NotificationChannel_Params_OneOf{}
	_ = o.FromTelegramParams(p)
	return o
}

func newOneOfWithWebhook(p generated.WebhookParams) *generated.NotificationChannel_Params_OneOf {
	o := &generated.NotificationChannel_Params_OneOf{}
	_ = o.FromWebhookParams(p)
	return o
}

func newOneOfWithAlertmanager(p generated.AlertmanagerParams) *generated.NotificationChannel_Params_OneOf {
	o := &generated.NotificationChannel_Params_OneOf{}
	_ = o.FromAlertmanagerParams(p)
	return o
}

func TestChannelToModel_slackWebhook(t *testing.T) {
	ch := &generated.NotificationChannel{
		ID:         100,
		ProjectID:  7,
		Name:       "alerts-slack",
		Type:       generated.Slack,
		Status:     generated.Delivering,
		MatchAll:   runtime.Ptr(true),
		Priorities: []generated.NotificationChannelPriorities{generated.NotificationChannelPrioritiesHigh},
		Params: generated.NotificationChannel_Params{
			NotificationChannel_Params_OneOf: newOneOfWithSlack(generated.SlackParams{
				AuthMethod: (*generated.SlackParamsAuthMethod)(runtime.Ptr("webhook")),
				WebhookURL: runtime.Ptr("https://hooks.slack.com/test"),
			}),
		},
	}
	m := notificationChannelModel{ProjectID: types.StringValue("7")}

	diags := channelToModel(context.Background(), ch, &m)
	require.False(t, diags.HasError(), "channelToModel returned errors: %v", diags)

	require.Equal(t, types.StringValue("100"), m.ID)
	require.Equal(t, types.StringValue("7"), m.ProjectID, "ProjectID must not be modified by the mapper")
	require.Equal(t, types.StringValue("alerts-slack"), m.Name)
	require.Equal(t, types.StringValue("slack"), m.Type)
	require.Equal(t, types.StringValue("delivering"), m.Status)
	require.Equal(t, types.BoolValue(true), m.MatchAll)
	require.NotNil(t, m.Slack)
	require.Equal(t, types.StringValue("webhook"), m.Slack.AuthMethod)
	require.Equal(t, types.StringValue("https://hooks.slack.com/test"), m.Slack.WebhookURL)
}

func TestChannelToModel_slackDoesNotPreserveMissingAuthMethod(t *testing.T) {
	ch := &generated.NotificationChannel{
		ID:         101,
		ProjectID:  7,
		Name:       "alerts-slack",
		Type:       generated.Slack,
		Status:     generated.Delivering,
		MatchAll:   runtime.Ptr(true),
		Priorities: []generated.NotificationChannelPriorities{generated.NotificationChannelPrioritiesHigh},
		Params: generated.NotificationChannel_Params{
			NotificationChannel_Params_OneOf: newOneOfWithSlack(generated.SlackParams{
				AuthMethod: nil,
				WebhookURL: runtime.Ptr("https://hooks.slack.com/test"),
			}),
		},
	}
	m := notificationChannelModel{
		Slack: &slackModel{
			AuthMethod: types.StringValue("webhook"),
		},
	}

	diags := channelToModel(context.Background(), ch, &m)
	require.False(t, diags.HasError(), "channelToModel returned errors: %v", diags)

	require.NotNil(t, m.Slack)
	require.True(t, m.Slack.AuthMethod.IsNull())
}

func TestChannelToModel_telegram(t *testing.T) {
	ch := &generated.NotificationChannel{
		ID:         200,
		ProjectID:  7,
		Name:       "alerts-tg",
		Type:       generated.Telegram,
		Status:     generated.Delivering,
		MatchAll:   runtime.Ptr(true),
		Priorities: []generated.NotificationChannelPriorities{generated.NotificationChannelPrioritiesHigh},
		Params: generated.NotificationChannel_Params{
			NotificationChannel_Params_OneOf: newOneOfWithTelegram(generated.TelegramParams{
				ChatID: -100123456,
			}),
		},
	}
	var m notificationChannelModel

	diags := channelToModel(context.Background(), ch, &m)
	require.False(t, diags.HasError(), "channelToModel returned errors: %v", diags)

	require.NotNil(t, m.Telegram)
	require.Equal(t, types.Int64Value(-100123456), m.Telegram.ChatID)
	require.Nil(t, m.Slack)
	require.Nil(t, m.Webhook)
}

func TestChannelToModel_webhook(t *testing.T) {
	ch := &generated.NotificationChannel{
		ID:         300,
		ProjectID:  7,
		Name:       "alerts-hook",
		Type:       generated.NotificationChannelTypeWebhook,
		Status:     generated.Delivering,
		MatchAll:   runtime.Ptr(false),
		Condition:  runtime.Ptr("severity >= warning"),
		Priorities: []generated.NotificationChannelPriorities{generated.NotificationChannelPrioritiesHigh, generated.NotificationChannelPrioritiesLow},
		MonitorIds: []int64{1, 2, 3},
		Params: generated.NotificationChannel_Params{
			NotificationChannel_Params_OneOf: newOneOfWithWebhook(generated.WebhookParams{
				URL: "https://example.com/hook",
			}),
		},
	}
	var m notificationChannelModel

	diags := channelToModel(context.Background(), ch, &m)
	require.False(t, diags.HasError(), "channelToModel returned errors: %v", diags)

	require.Equal(t, types.BoolValue(false), m.MatchAll)
	require.Equal(t, types.StringValue("severity >= warning"), m.Condition)
	require.NotNil(t, m.Webhook)
	require.Equal(t, types.StringValue("https://example.com/hook"), m.Webhook.URL)
	require.False(t, m.MonitorIDs.IsNull())
}

func TestChannelToModel_webhookWithPayload(t *testing.T) {
	ch := &generated.NotificationChannel{
		ID:         301,
		ProjectID:  7,
		Name:       "alerts-hook",
		Type:       generated.NotificationChannelTypeWebhook,
		Status:     generated.Delivering,
		MatchAll:   runtime.Ptr(true),
		Priorities: []generated.NotificationChannelPriorities{generated.NotificationChannelPrioritiesHigh},
		Params: generated.NotificationChannel_Params{
			NotificationChannel_Params_OneOf: newOneOfWithWebhook(generated.WebhookParams{
				URL: "https://example.com/hook",
				Payload: map[string]any{
					"alert":    "uptrace",
					"severity": "high",
				},
			}),
		},
	}
	var m notificationChannelModel

	diags := channelToModel(context.Background(), ch, &m)
	require.False(t, diags.HasError(), "channelToModel returned errors: %v", diags)
	require.NotNil(t, m.Webhook)
	require.False(t, m.Webhook.Payload.IsNull())
	// json.Marshal of map[string]any sorts keys, so the output is stable.
	require.Equal(t, `{"alert":"uptrace","severity":"high"}`, m.Webhook.Payload.ValueString())
}

func TestChannelToModel_webhookEmptyPayload(t *testing.T) {
	ch := &generated.NotificationChannel{
		ID:         302,
		ProjectID:  7,
		Name:       "alerts-hook",
		Type:       generated.NotificationChannelTypeWebhook,
		Status:     generated.Delivering,
		MatchAll:   runtime.Ptr(true),
		Priorities: []generated.NotificationChannelPriorities{generated.NotificationChannelPrioritiesHigh},
		Params: generated.NotificationChannel_Params{
			NotificationChannel_Params_OneOf: newOneOfWithWebhook(generated.WebhookParams{
				URL: "https://example.com/hook",
			}),
		},
	}
	var m notificationChannelModel

	diags := channelToModel(context.Background(), ch, &m)
	require.False(t, diags.HasError(), "channelToModel returned errors: %v", diags)
	require.NotNil(t, m.Webhook)
	require.True(t, m.Webhook.Payload.IsNull())
}

func TestChannelToModel_preservesExplicitEmptyMonitorIDs(t *testing.T) {
	emptyMonitorIDs, d := types.ListValueFrom(context.Background(), types.StringType, []types.String{})
	require.False(t, d.HasError(), "ListValueFrom returned errors: %v", d)
	ch := &generated.NotificationChannel{
		ID:         303,
		ProjectID:  7,
		Name:       "alerts-hook",
		Type:       generated.NotificationChannelTypeWebhook,
		Status:     generated.Delivering,
		MatchAll:   runtime.Ptr(true),
		Priorities: []generated.NotificationChannelPriorities{generated.NotificationChannelPrioritiesHigh},
		Params: generated.NotificationChannel_Params{
			NotificationChannel_Params_OneOf: newOneOfWithWebhook(generated.WebhookParams{
				URL: "https://example.com/hook",
			}),
		},
	}
	m := notificationChannelModel{MonitorIDs: emptyMonitorIDs}

	diags := channelToModel(context.Background(), ch, &m)
	require.False(t, diags.HasError(), "channelToModel returned errors: %v", diags)

	require.False(t, m.MonitorIDs.IsNull())
	require.Len(t, m.MonitorIDs.Elements(), 0)
}

func TestChannelToModel_doesNotSetProjectID(t *testing.T) {
	ch := &generated.NotificationChannel{
		ID:         100,
		ProjectID:  7,
		Name:       "test",
		Type:       generated.Telegram,
		Status:     generated.Delivering,
		Priorities: []generated.NotificationChannelPriorities{generated.NotificationChannelPrioritiesHigh},
		Params: generated.NotificationChannel_Params{
			NotificationChannel_Params_OneOf: newOneOfWithTelegram(generated.TelegramParams{ChatID: 1}),
		},
	}
	m := notificationChannelModel{ProjectID: types.StringValue("99")}

	diags := channelToModel(context.Background(), ch, &m)
	require.False(t, diags.HasError(), "channelToModel returned errors: %v", diags)

	require.Equal(t, types.StringValue("99"), m.ProjectID,
		"mapper must not overwrite ProjectID — callers own that field")
}

func TestChannelToModel_nilConditionBecomesNull(t *testing.T) {
	ch := &generated.NotificationChannel{
		ID:         100,
		ProjectID:  7,
		Name:       "test",
		Type:       generated.Telegram,
		Status:     generated.Delivering,
		Priorities: []generated.NotificationChannelPriorities{generated.NotificationChannelPrioritiesHigh},
		Params: generated.NotificationChannel_Params{
			NotificationChannel_Params_OneOf: newOneOfWithTelegram(generated.TelegramParams{ChatID: 1}),
		},
	}
	var m notificationChannelModel

	diags := channelToModel(context.Background(), ch, &m)
	require.False(t, diags.HasError(), "channelToModel returned errors: %v", diags)

	require.True(t, m.Condition.IsNull())
}

func TestChannelToModel_alertmanagerPreservesRedactedSensitiveValues(t *testing.T) {
	ch := &generated.NotificationChannel{
		ID:         400,
		ProjectID:  7,
		Name:       "alerts-am",
		Type:       generated.Alertmanager,
		Status:     generated.Delivering,
		Priorities: []generated.NotificationChannelPriorities{generated.NotificationChannelPrioritiesHigh},
		Params: generated.NotificationChannel_Params{
			NotificationChannel_Params_OneOf: newOneOfWithAlertmanager(generated.AlertmanagerParams{
				URL:        "https://alertmanager.example.com",
				AuthMethod: (*generated.AlertmanagerParamsAuthMethod)(runtime.Ptr("bearer")),
				Password:   nil,
				Token:      runtime.Ptr(""),
			}),
		},
	}
	m := notificationChannelModel{
		Alertmanager: &alertmanagerModel{
			Password: types.StringValue("prior-password"),
			Token:    types.StringValue("prior-token"),
		},
	}

	diags := channelToModel(context.Background(), ch, &m)
	require.False(t, diags.HasError(), "channelToModel returned errors: %v", diags)

	require.NotNil(t, m.Alertmanager)
	require.Equal(t, types.StringValue("prior-password"), m.Alertmanager.Password)
	require.Equal(t, types.StringValue("prior-token"), m.Alertmanager.Token)
}

func TestChannelToModel_servicenowPreservesRedactedUsername(t *testing.T) {
	oneOf := &generated.NotificationChannel_Params_OneOf{}
	require.NoError(t, oneOf.UnmarshalJSON([]byte(`{
		"url": "https://servicenow.example.com",
		"username": "",
		"password": ""
	}`)))
	ch := &generated.NotificationChannel{
		ID:         500,
		ProjectID:  7,
		Name:       "alerts-servicenow",
		Type:       generated.Servicenow,
		Status:     generated.Delivering,
		Priorities: []generated.NotificationChannelPriorities{generated.NotificationChannelPrioritiesHigh},
		Params: generated.NotificationChannel_Params{
			NotificationChannel_Params_OneOf: oneOf,
		},
	}
	m := notificationChannelModel{
		Servicenow: &servicenowModel{
			Username: types.StringValue("prior-user"),
		},
	}

	diags := channelToModel(context.Background(), ch, &m)
	require.False(t, diags.HasError(), "channelToModel returned errors: %v", diags)

	require.NotNil(t, m.Servicenow)
	require.Equal(t, types.StringValue("prior-user"), m.Servicenow.Username)
}

func TestSchema_opsgeniePriorityRejectsInvalidValues(t *testing.T) {
	var resp resource.SchemaResponse
	NewNotificationChannelResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "Schema returned errors: %v", resp.Diagnostics)

	opsgenieBlock, ok := resp.Schema.Blocks["opsgenie"].(rschema.SingleNestedBlock)
	require.True(t, ok, "opsgenie block should be a single nested block")
	priorityAttr, ok := opsgenieBlock.Attributes["priority"].(rschema.StringAttribute)
	require.True(t, ok, "opsgenie.priority should be a string attribute")
	require.NotEmpty(t, priorityAttr.Validators, "opsgenie.priority should validate enum values")

	require.False(t, validateOpsgeniePriority(priorityAttr.Validators, "P1").Diagnostics.HasError())
	require.True(t, validateOpsgeniePriority(priorityAttr.Validators, "P0").Diagnostics.HasError())
}

func validateOpsgeniePriority(validators []validator.String, value string) validator.StringResponse {
	resp := validator.StringResponse{}
	for _, v := range validators {
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("opsgenie").AtName("priority"),
			ConfigValue: types.StringValue(value),
		}, &resp)
	}
	return resp
}

func TestParseChannelID_valid(t *testing.T) {
	id, err := parseChannelID("123")
	require.NoError(t, err)
	require.Equal(t, int64(123), id)
}

func TestParseChannelID_invalid(t *testing.T) {
	_, err := parseChannelID("abc")
	require.Error(t, err)
}
