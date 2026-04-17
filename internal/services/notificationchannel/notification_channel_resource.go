package notificationchannel

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/client"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
)

var (
	_ resource.Resource                   = &NotificationChannelResource{}
	_ resource.ResourceWithConfigure      = &NotificationChannelResource{}
	_ resource.ResourceWithImportState    = &NotificationChannelResource{}
	_ resource.ResourceWithValidateConfig = &NotificationChannelResource{}
)

type NotificationChannelResource struct {
	client *client.Client
}

// notificationChannelModel maps the Terraform state for uptrace_notification_channel.
type notificationChannelModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Status    types.String `tfsdk:"status"`
	MatchAll  types.Bool   `tfsdk:"match_all"`
	Condition types.String `tfsdk:"condition"`

	Priorities types.List `tfsdk:"priorities"`
	MonitorIDs types.List `tfsdk:"monitor_ids"`

	// Type-specific param blocks (exactly one must be set).
	Slack        *slackModel        `tfsdk:"slack"`
	GoogleChat   *googleChatModel   `tfsdk:"google_chat"`
	Mattermost   *mattermostModel   `tfsdk:"mattermost"`
	Pagerduty    *pagerdutyModel    `tfsdk:"pagerduty"`
	Servicenow   *servicenowModel   `tfsdk:"servicenow"`
	Opsgenie     *opsgenieModel     `tfsdk:"opsgenie"`
	Telegram     *telegramModel     `tfsdk:"telegram"`
	Teams        *teamsModel        `tfsdk:"teams"`
	Pushover     *pushoverModel     `tfsdk:"pushover"`
	Webhook      *webhookModel      `tfsdk:"webhook"`
	Alertmanager *alertmanagerModel `tfsdk:"alertmanager"`
	Incidentio   *incidentioModel   `tfsdk:"incidentio"`
}

// --- param models ---

type slackModel struct {
	AuthMethod types.String `tfsdk:"auth_method"`
	WebhookURL types.String `tfsdk:"webhook_url"`
	Token      types.String `tfsdk:"token"`
	Channel    types.String `tfsdk:"channel"`
}

type googleChatModel struct {
	WebhookURL types.String `tfsdk:"webhook_url"`
}

type mattermostModel struct {
	WebhookURL types.String `tfsdk:"webhook_url"`
}

type pagerdutyModel struct {
	RoutingKey types.String `tfsdk:"routing_key"`
	Severity   types.String `tfsdk:"severity"`
}

type servicenowModel struct {
	URL         types.String `tfsdk:"url"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	Category    types.String `tfsdk:"category"`
	Subcategory types.String `tfsdk:"subcategory"`
	Impact      types.String `tfsdk:"impact"`
	Urgency     types.String `tfsdk:"urgency"`
	Severity    types.String `tfsdk:"severity"`
	CallerID    types.String `tfsdk:"caller_id"`
	Group       types.String `tfsdk:"group"`
	AssignedTo  types.String `tfsdk:"assigned_to"`
	OpenedBy    types.String `tfsdk:"opened_by"`
	Notify      types.String `tfsdk:"notify"`
	DueDate     types.String `tfsdk:"due_date"`
}

type opsgenieModel struct {
	APIKey   types.String `tfsdk:"api_key"`
	Priority types.String `tfsdk:"priority"`
}

type telegramModel struct {
	ChatID types.Int64 `tfsdk:"chat_id"`
}

type teamsModel struct {
	WebhookURL types.String `tfsdk:"webhook_url"`
}

type pushoverModel struct {
	Token    types.String `tfsdk:"token"`
	UserKey  types.String `tfsdk:"user_key"`
	Priority types.Int64  `tfsdk:"priority"`
	Sound    types.String `tfsdk:"sound"`
}

type webhookModel struct {
	URL     types.String `tfsdk:"url"`
	Payload types.String `tfsdk:"payload"`
}

type alertmanagerModel struct {
	URL        types.String `tfsdk:"url"`
	AuthMethod types.String `tfsdk:"auth_method"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	Token      types.String `tfsdk:"token"`
}

type incidentioModel struct {
	URL    types.String `tfsdk:"url"`
	APIKey types.String `tfsdk:"api_key"`
}

func NewNotificationChannelResource() resource.Resource {
	return &NotificationChannelResource{}
}

func (r *NotificationChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_channel"
}

func (r *NotificationChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	channelTypes := []string{
		"slack", "google_chat", "mattermost", "pagerduty", "servicenow",
		"opsgenie", "telegram", "teams", "pushover", "webhook",
		"alertmanager", "incidentio",
	}
	priorities := []string{"info", "low", "medium", "high"}
	pagerdutySeverities := []string{"critical", "error", "warning", "info"}
	servicenowImpacts := []string{"1", "2", "3"}
	servicenowUrgencies := []string{"1", "2", "3"}
	servicenowSeverities := []string{"1", "2", "3", "4", "5"}
	servicenowNotifies := []string{"1", "2"}

	resp.Schema = schema.Schema{
		Description: "Manages an Uptrace notification channel.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Channel ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "Project ID this channel belongs to. Changing this forces recreation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable channel name.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Notification channel type.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(channelTypes...),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Channel status (delivering, paused, disabled, draft).",
			},
			"match_all": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Match all monitors. If false, monitor_ids must be provided.",
			},
			"condition": schema.StringAttribute{
				Optional:    true,
				Description: "Alert condition expression.",
			},
			"priorities": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: fmt.Sprintf("Alert priorities to match. Valid values: %s.", strings.Join(priorities, ", ")),
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(priorities...)),
				},
			},
			"monitor_ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Monitor IDs to match when match_all is false.",
			},
		},
		Blocks: map[string]schema.Block{
			"slack": schema.SingleNestedBlock{
				Description: "Slack channel params. Exactly one param block must match the type.",
				Attributes: map[string]schema.Attribute{
					"auth_method": schema.StringAttribute{
						Optional:    true,
						Description: "Authentication method: webhook or token.",
						Validators: []validator.String{
							stringvalidator.OneOf("webhook", "token"),
						},
					},
					"webhook_url": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Slack webhook URL (when auth_method is webhook).",
					},
					"token": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Slack bot token (when auth_method is token).",
					},
					"channel": schema.StringAttribute{
						Optional:    true,
						Description: "Slack channel or user (when auth_method is token).",
					},
				},
			},
			"google_chat": schema.SingleNestedBlock{
				Description: "Google Chat params.",
				Attributes: map[string]schema.Attribute{
					"webhook_url": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Google Chat webhook URL.",
					},
				},
			},
			"mattermost": schema.SingleNestedBlock{
				Description: "Mattermost params.",
				Attributes: map[string]schema.Attribute{
					"webhook_url": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Mattermost webhook URL.",
					},
				},
			},
			"pagerduty": schema.SingleNestedBlock{
				Description: "PagerDuty params.",
				Attributes: map[string]schema.Attribute{
					"routing_key": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "PagerDuty routing key.",
					},
					"severity": schema.StringAttribute{
						Optional:    true,
						Description: fmt.Sprintf("PagerDuty severity. Valid values: %s.", strings.Join(pagerdutySeverities, ", ")),
						Validators: []validator.String{
							stringvalidator.OneOf(pagerdutySeverities...),
						},
					},
				},
			},
			"servicenow": schema.SingleNestedBlock{
				Description: "ServiceNow params.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "ServiceNow instance URL.",
					},
					"username": schema.StringAttribute{
						Optional:    true,
						Description: "ServiceNow username.",
					},
					"password": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "ServiceNow password.",
					},
					"category": schema.StringAttribute{
						Optional:    true,
						Description: "Incident category.",
					},
					"subcategory": schema.StringAttribute{
						Optional:    true,
						Description: "Incident subcategory.",
					},
					"impact": schema.StringAttribute{
						Optional:    true,
						Description: "Incident impact (1, 2, or 3).",
						Validators: []validator.String{
							stringvalidator.OneOf(servicenowImpacts...),
						},
					},
					"urgency": schema.StringAttribute{
						Optional:    true,
						Description: "Incident urgency (1, 2, or 3).",
						Validators: []validator.String{
							stringvalidator.OneOf(servicenowUrgencies...),
						},
					},
					"severity": schema.StringAttribute{
						Optional:    true,
						Description: "Incident severity (1-5).",
						Validators: []validator.String{
							stringvalidator.OneOf(servicenowSeverities...),
						},
					},
					"caller_id": schema.StringAttribute{
						Optional:    true,
						Description: "Caller ID.",
					},
					"group": schema.StringAttribute{
						Optional:    true,
						Description: "Assignment group.",
					},
					"assigned_to": schema.StringAttribute{
						Optional:    true,
						Description: "Assigned to user.",
					},
					"opened_by": schema.StringAttribute{
						Optional:    true,
						Description: "Opened by user.",
					},
					"notify": schema.StringAttribute{
						Optional:    true,
						Description: "Notify setting (1 or 2).",
						Validators: []validator.String{
							stringvalidator.OneOf(servicenowNotifies...),
						},
					},
					"due_date": schema.StringAttribute{
						Optional:    true,
						Description: "Due date.",
					},
				},
			},
			"opsgenie": schema.SingleNestedBlock{
				Description: "Opsgenie params.",
				Attributes: map[string]schema.Attribute{
					"api_key": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Opsgenie API key.",
					},
					"priority": schema.StringAttribute{
						Optional:    true,
						Description: "Opsgenie alert priority.",
						Validators: []validator.String{
							stringvalidator.OneOf("P1", "P2", "P3", "P4", "P5"),
						},
					},
				},
			},
			"telegram": schema.SingleNestedBlock{
				Description: "Telegram params.",
				Attributes: map[string]schema.Attribute{
					"chat_id": schema.Int64Attribute{
						Optional:    true,
						Description: "Telegram chat ID.",
					},
				},
			},
			"teams": schema.SingleNestedBlock{
				Description: "Microsoft Teams params.",
				Attributes: map[string]schema.Attribute{
					"webhook_url": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Teams webhook URL.",
					},
				},
			},
			"pushover": schema.SingleNestedBlock{
				Description: "Pushover params.",
				Attributes: map[string]schema.Attribute{
					"token": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Pushover application token.",
					},
					"user_key": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Pushover user key.",
					},
					"priority": schema.Int64Attribute{
						Optional:    true,
						Description: "Priority from -2 (lowest) to 2 (emergency).",
					},
					"sound": schema.StringAttribute{
						Optional:    true,
						Description: "Notification sound.",
					},
				},
			},
			"webhook": schema.SingleNestedBlock{
				Description: "Webhook params.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Webhook URL.",
					},
					"payload": schema.StringAttribute{
						Optional:    true,
						Description: "Optional JSON object sent as the webhook payload. Use jsonencode() to construct.",
						Validators: []validator.String{
							jsonObjectValidator{},
						},
					},
				},
			},
			"alertmanager": schema.SingleNestedBlock{
				Description: "Alertmanager params.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Optional:    true,
						Description: "Alertmanager API endpoint URL.",
					},
					"auth_method": schema.StringAttribute{
						Optional:    true,
						Description: "Authentication method: none, basic_auth, or bearer.",
						Validators: []validator.String{
							stringvalidator.OneOf("none", "basic_auth", "bearer"),
						},
					},
					"username": schema.StringAttribute{
						Optional:    true,
						Description: "Username for basic_auth.",
					},
					"password": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Password for basic_auth.",
					},
					"token": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Token for bearer auth.",
					},
				},
			},
			"incidentio": schema.SingleNestedBlock{
				Description: "incident.io params.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Optional:    true,
						Description: "Alert Events V2 endpoint URL.",
					},
					"api_key": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "incident.io API key.",
					},
				},
			},
		},
	}
}

func (r *NotificationChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected provider data type",
			fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *NotificationChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan notificationChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := parseProjectID(plan.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid project_id", err.Error())
		return
	}

	body, diags := buildRequestBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "creating notification channel", map[string]any{
		"project_id": plan.ProjectID.ValueString(),
		"type":       plan.Type.ValueString(),
	})

	out, err := r.client.API.CreateNotificationChannel(ctx, &generated.CreateNotificationChannelRequestOptions{
		PathParams: &generated.CreateNotificationChannelPath{ProjectID: projectID},
		Body:       body,
	})
	if err != nil {
		resp.Diagnostics.AddError("create notification channel failed", err.Error())
		return
	}

	resp.Diagnostics.Append(channelToModel(ctx, &out.Channel, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NotificationChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state notificationChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := parseProjectID(state.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid project_id", err.Error())
		return
	}

	channelID, err := parseChannelID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid channel ID", err.Error())
		return
	}

	out, err := r.client.API.GetNotificationChannel(ctx, &generated.GetNotificationChannelRequestOptions{
		PathParams: &generated.GetNotificationChannelPath{
			ProjectID: projectID,
			ChannelID: channelID,
		},
	})
	if err != nil {
		if client.IsNotFound(err) || client.IsForbidden(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read notification channel failed", err.Error())
		return
	}

	resp.Diagnostics.Append(channelToModel(ctx, &out.Channel, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotificationChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan notificationChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := parseProjectID(plan.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid project_id", err.Error())
		return
	}

	channelID, err := parseChannelID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid channel ID", err.Error())
		return
	}

	body, diags := buildRequestBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "updating notification channel", map[string]any{
		"id": plan.ID.ValueString(),
	})

	out, err := r.client.API.UpdateNotificationChannel(ctx, &generated.UpdateNotificationChannelRequestOptions{
		PathParams: &generated.UpdateNotificationChannelPath{
			ProjectID: projectID,
			ChannelID: channelID,
		},
		Body: body,
	})
	if err != nil {
		resp.Diagnostics.AddError("update notification channel failed", err.Error())
		return
	}

	resp.Diagnostics.Append(channelToModel(ctx, &out.Channel, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NotificationChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state notificationChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := parseProjectID(state.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid project_id", err.Error())
		return
	}

	channelID, err := parseChannelID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid channel ID", err.Error())
		return
	}

	tflog.Info(ctx, "deleting notification channel", map[string]any{
		"id": state.ID.ValueString(),
	})

	_, err = r.client.API.DeleteNotificationChannel(ctx, &generated.DeleteNotificationChannelRequestOptions{
		PathParams: &generated.DeleteNotificationChannelPath{
			ProjectID: projectID,
			ChannelID: channelID,
		},
	})
	if err != nil && !client.IsNotFound(err) && !client.IsForbidden(err) {
		resp.Diagnostics.AddError("delete notification channel failed", err.Error())
	}
}

// ImportState accepts "<project_id>:<channel_id>".
func (r *NotificationChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"invalid import ID",
			fmt.Sprintf("expected format <project_id>:<channel_id>, got %q", req.ID),
		)
		return
	}
	if _, err := parseProjectID(parts[0]); err != nil {
		resp.Diagnostics.AddError("invalid project_id in import ID", err.Error())
		return
	}
	if _, err := parseChannelID(parts[1]); err != nil {
		resp.Diagnostics.AddError("invalid channel_id in import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func parseProjectID(s string) (uint32, error) {
	return client.ParseProjectID(s)
}

func parseChannelID(s string) (int64, error) {
	return client.ParseChannelID(s)
}

// channelToModel maps the API response to the Terraform model.
// It does NOT overwrite ProjectID — callers own that field.
func channelToModel(ctx context.Context, ch *generated.NotificationChannel, m *notificationChannelModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(strconv.FormatInt(ch.ID, 10))
	m.Name = types.StringValue(ch.Name)
	m.Type = types.StringValue(string(ch.Type))
	m.Status = types.StringValue(string(ch.Status))

	if ch.MatchAll != nil {
		m.MatchAll = types.BoolValue(*ch.MatchAll)
	} else {
		m.MatchAll = types.BoolValue(true)
	}

	if ch.Condition != nil && *ch.Condition != "" {
		m.Condition = types.StringValue(*ch.Condition)
	} else {
		m.Condition = types.StringNull()
	}

	// Priorities.
	if len(ch.Priorities) > 0 {
		vals := make([]types.String, len(ch.Priorities))
		for i, p := range ch.Priorities {
			vals[i] = types.StringValue(string(p))
		}
		var d diag.Diagnostics
		m.Priorities, d = types.ListValueFrom(ctx, types.StringType, vals)
		diags.Append(d...)
	} else {
		var d diag.Diagnostics
		m.Priorities, d = types.ListValueFrom(ctx, types.StringType, []types.String{})
		diags.Append(d...)
	}

	// MonitorIDs. Preserve an explicit empty list from prior state so a user
	// writing `monitor_ids = []` round-trips instead of flipping to null.
	switch {
	case len(ch.MonitorIds) > 0:
		vals := make([]types.String, len(ch.MonitorIds))
		for i, id := range ch.MonitorIds {
			vals[i] = types.StringValue(strconv.FormatInt(id, 10))
		}
		var d diag.Diagnostics
		m.MonitorIDs, d = types.ListValueFrom(ctx, types.StringType, vals)
		diags.Append(d...)
	case !m.MonitorIDs.IsNull() && !m.MonitorIDs.IsUnknown() && len(m.MonitorIDs.Elements()) == 0:
		// Prior was an explicit empty list; keep it.
	default:
		m.MonitorIDs = types.ListNull(types.StringType)
	}

	// Params — map into the correct nested block based on type.
	if ch.Params.NotificationChannel_Params_OneOf != nil {
		diags.Append(paramsToModel(ctx, ch, m)...)
	}

	return diags
}

func paramsToModel(ctx context.Context, ch *generated.NotificationChannel, m *notificationChannelModel) diag.Diagnostics {
	var diags diag.Diagnostics
	oneOf := ch.Params.NotificationChannel_Params_OneOf

	warnMalformedParams := func(channelType string, err error) {
		tflog.Warn(ctx, "failed to decode notification channel params", map[string]any{
			"channel_type": channelType,
			"error":        err.Error(),
		})
		diags.AddWarning(
			"failed to decode notification channel params",
			fmt.Sprintf("could not decode %s params: %s", channelType, err.Error()),
		)
	}

	// Fields marked Sensitive in the schema route through preserveIfEmpty so a
	// redacted or empty API response falls back to the prior state value; other
	// fields overwrite directly so genuine out-of-band drift is visible.
	switch ch.Type {
	case generated.Slack:
		p, err := oneOf.AsSlackParams()
		if err != nil {
			warnMalformedParams("slack", err)
			return diags
		}
		prior := m.Slack
		authMethod := types.StringNull()
		if p.AuthMethod != nil {
			authMethod = types.StringValue(string(*p.AuthMethod))
		}
		m.Slack = &slackModel{
			AuthMethod: authMethod,
			WebhookURL: preserveIfEmptyPtr(p.WebhookURL, priorString(prior, func(x *slackModel) types.String { return x.WebhookURL })),
			Token:      preserveIfEmptyPtr(p.Token, priorString(prior, func(x *slackModel) types.String { return x.Token })),
			Channel:    preserveIfEmptyPtr(p.Channel, priorString(prior, func(x *slackModel) types.String { return x.Channel })),
		}

	case generated.GoogleChat:
		p, err := oneOf.AsGoogleChatParams()
		if err != nil {
			warnMalformedParams("google_chat", err)
			return diags
		}
		prior := m.GoogleChat
		m.GoogleChat = &googleChatModel{
			WebhookURL: preserveIfEmpty(p.WebhookURL, priorString(prior, func(x *googleChatModel) types.String { return x.WebhookURL })),
		}

	case generated.Mattermost:
		p, err := oneOf.AsMattermostParams()
		if err != nil {
			warnMalformedParams("mattermost", err)
			return diags
		}
		prior := m.Mattermost
		m.Mattermost = &mattermostModel{
			WebhookURL: preserveIfEmpty(p.WebhookURL, priorString(prior, func(x *mattermostModel) types.String { return x.WebhookURL })),
		}

	case generated.Pagerduty:
		p, err := oneOf.AsPagerdutyParams()
		if err != nil {
			warnMalformedParams("pagerduty", err)
			return diags
		}
		prior := m.Pagerduty
		m.Pagerduty = &pagerdutyModel{
			RoutingKey: preserveIfEmpty(p.RoutingKey, priorString(prior, func(x *pagerdutyModel) types.String { return x.RoutingKey })),
			Severity:   types.StringValue(string(p.Severity)),
		}

	case generated.Servicenow:
		p, err := oneOf.AsServicenowParams()
		if err != nil {
			warnMalformedParams("servicenow", err)
			return diags
		}
		prior := m.Servicenow
		sm := &servicenowModel{
			URL:      preserveIfEmpty(p.URL, priorString(prior, func(x *servicenowModel) types.String { return x.URL })),
			Username: preserveIfEmpty(p.Username, priorString(prior, func(x *servicenowModel) types.String { return x.Username })),
			Password: preserveIfEmpty(p.Password, priorString(prior, func(x *servicenowModel) types.String { return x.Password })),
		}
		sm.Category = optionalStringToValue(p.Category)
		sm.Subcategory = optionalStringToValue(p.Subcategory)
		sm.Impact = enumToValue(p.Impact)
		sm.Urgency = enumToValue(p.Urgency)
		sm.Severity = enumToValue(p.Severity)
		sm.CallerID = optionalStringToValue(p.CallerID)
		sm.Group = optionalStringToValue(p.Group)
		sm.AssignedTo = optionalStringToValue(p.AssignedTo)
		sm.OpenedBy = optionalStringToValue(p.OpenedBy)
		sm.Notify = enumToValue(p.Notify)
		sm.DueDate = optionalStringToValue(p.DueDate)
		m.Servicenow = sm

	case generated.Opsgenie:
		p, err := oneOf.AsOpsgenieParams()
		if err != nil {
			warnMalformedParams("opsgenie", err)
			return diags
		}
		prior := m.Opsgenie
		m.Opsgenie = &opsgenieModel{
			APIKey:   preserveIfEmpty(p.APIKey, priorString(prior, func(x *opsgenieModel) types.String { return x.APIKey })),
			Priority: types.StringValue(string(p.Priority)),
		}

	case generated.Telegram:
		p, err := oneOf.AsTelegramParams()
		if err != nil {
			warnMalformedParams("telegram", err)
			return diags
		}
		m.Telegram = &telegramModel{
			ChatID: types.Int64Value(p.ChatID),
		}

	case generated.Teams:
		p, err := oneOf.AsTeamsParams()
		if err != nil {
			warnMalformedParams("teams", err)
			return diags
		}
		prior := m.Teams
		m.Teams = &teamsModel{
			WebhookURL: preserveIfEmpty(p.WebhookURL, priorString(prior, func(x *teamsModel) types.String { return x.WebhookURL })),
		}

	case generated.Pushover:
		p, err := oneOf.AsPushoverParams()
		if err != nil {
			warnMalformedParams("pushover", err)
			return diags
		}
		prior := m.Pushover
		m.Pushover = &pushoverModel{
			Token:    preserveIfEmpty(p.Token, priorString(prior, func(x *pushoverModel) types.String { return x.Token })),
			UserKey:  preserveIfEmpty(p.UserKey, priorString(prior, func(x *pushoverModel) types.String { return x.UserKey })),
			Priority: intPtrToValue(p.Priority, priorInt64(prior, func(x *pushoverModel) types.Int64 { return x.Priority })),
			Sound:    optionalStringToValue(p.Sound),
		}

	case generated.NotificationChannelTypeWebhook:
		p, err := oneOf.AsWebhookParams()
		if err != nil {
			warnMalformedParams("webhook", err)
			return diags
		}
		prior := m.Webhook
		wm := &webhookModel{
			URL:     preserveIfEmpty(p.URL, priorString(prior, func(x *webhookModel) types.String { return x.URL })),
			Payload: types.StringNull(),
		}
		if len(p.Payload) > 0 {
			buf, err := json.Marshal(p.Payload)
			if err != nil {
				diags.AddWarning(
					"failed to encode webhook payload",
					fmt.Sprintf("could not marshal webhook payload to JSON: %s", err.Error()),
				)
			} else {
				wm.Payload = types.StringValue(string(buf))
			}
		}
		m.Webhook = wm

	case generated.Alertmanager:
		p, err := oneOf.AsAlertmanagerParams()
		if err != nil {
			warnMalformedParams("alertmanager", err)
			return diags
		}
		prior := m.Alertmanager
		m.Alertmanager = &alertmanagerModel{
			URL:        types.StringValue(p.URL),
			AuthMethod: enumToValue(p.AuthMethod),
			Username:   optionalStringToValue(p.Username),
			Password:   preserveIfEmptyPtr(p.Password, priorString(prior, func(x *alertmanagerModel) types.String { return x.Password })),
			Token:      preserveIfEmptyPtr(p.Token, priorString(prior, func(x *alertmanagerModel) types.String { return x.Token })),
		}

	case generated.Incidentio:
		p, err := oneOf.AsIncidentioParams()
		if err != nil {
			warnMalformedParams("incidentio", err)
			return diags
		}
		prior := m.Incidentio
		m.Incidentio = &incidentioModel{
			URL:    types.StringValue(p.URL),
			APIKey: preserveIfEmpty(p.APIKey, priorString(prior, func(x *incidentioModel) types.String { return x.APIKey })),
		}

	default:
		tflog.Warn(ctx, "unrecognized notification channel type in API response", map[string]any{
			"channel_type": string(ch.Type),
		})
	}

	return diags
}

func optionalStringToValue(s *string) types.String {
	if s != nil && *s != "" {
		return types.StringValue(*s)
	}
	return types.StringNull()
}

func preserveIfEmpty(api string, prior types.String) types.String {
	if api == "" {
		return prior
	}
	return types.StringValue(api)
}

// preserveIfEmptyPtr is preserveIfEmpty for *string-shaped API fields, where
// the backend distinguishes nil-pointer from pointer-to-empty but neither
// should clobber the prior state value.
func preserveIfEmptyPtr(api *string, prior types.String) types.String {
	if api == nil || *api == "" {
		return prior
	}
	return types.StringValue(*api)
}

func priorString[T any](prior *T, get func(*T) types.String) types.String {
	if prior == nil {
		return types.StringNull()
	}
	return get(prior)
}

func priorInt64[T any](prior *T, get func(*T) types.Int64) types.Int64 {
	if prior == nil {
		return types.Int64Null()
	}
	return get(prior)
}

// enumToValue maps a *Enum (string-backed) API field to types.String. nil or
// empty-string pointer collapses to Null so a plan-null value matches
// post-apply state even when the backend echoes "" for an unset enum.
func enumToValue[T ~string](p *T) types.String {
	if p == nil || *p == "" {
		return types.StringNull()
	}
	return types.StringValue(string(*p))
}

// intPtrToValue maps a *int API field to types.Int64. A nil or zero pointer
// with a null prior collapses to Null (matching a plan where the user did not
// set the field); otherwise the API value surfaces so out-of-band drift to a
// non-zero value is still visible.
func intPtrToValue(api *int, prior types.Int64) types.Int64 {
	if api == nil {
		return prior
	}
	if *api == 0 && prior.IsNull() {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*api))
}

// jsonObjectValidator ensures a string attribute parses as a JSON object
// (not a scalar, array, or malformed JSON).
type jsonObjectValidator struct{}

func (jsonObjectValidator) Description(context.Context) string {
	return "must be a JSON object (e.g. constructed with jsonencode())"
}

func (jsonObjectValidator) MarkdownDescription(context.Context) string {
	return "must be a JSON object (e.g. constructed with `jsonencode()`)"
}

func (jsonObjectValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var v any
	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &v); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "invalid JSON", err.Error())
		return
	}
	if _, ok := v.(map[string]any); !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"payload must be a JSON object",
			fmt.Sprintf("got %T; use a JSON object at the top level (e.g. jsonencode({ key = \"value\" }))", v),
		)
	}
}
