package notificationchannel

import (
	"context"
	"testing"

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

func TestParseChannelID_valid(t *testing.T) {
	id, err := parseChannelID("123")
	require.NoError(t, err)
	require.Equal(t, int64(123), id)
}

func TestParseChannelID_invalid(t *testing.T) {
	_, err := parseChannelID("abc")
	require.Error(t, err)
}
