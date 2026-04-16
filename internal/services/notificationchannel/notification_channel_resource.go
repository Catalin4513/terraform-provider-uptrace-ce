package notificationchannel

import (
	"context"
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
	_ resource.Resource                = &NotificationChannelResource{}
	_ resource.ResourceWithConfigure   = &NotificationChannelResource{}
	_ resource.ResourceWithImportState = &NotificationChannelResource{}
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
	URL types.String `tfsdk:"url"`
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
						Required:    true,
						Sensitive:   true,
						Description: "Google Chat webhook URL.",
					},
				},
			},
			"mattermost": schema.SingleNestedBlock{
				Description: "Mattermost params.",
				Attributes: map[string]schema.Attribute{
					"webhook_url": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "Mattermost webhook URL.",
					},
				},
			},
			"pagerduty": schema.SingleNestedBlock{
				Description: "PagerDuty params.",
				Attributes: map[string]schema.Attribute{
					"routing_key": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "PagerDuty routing key.",
					},
					"severity": schema.StringAttribute{
						Required:    true,
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
						Required:    true,
						Sensitive:   true,
						Description: "ServiceNow instance URL.",
					},
					"username": schema.StringAttribute{
						Required:    true,
						Description: "ServiceNow username.",
					},
					"password": schema.StringAttribute{
						Required:    true,
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
						Required:    true,
						Sensitive:   true,
						Description: "Opsgenie API key.",
					},
					"priority": schema.StringAttribute{
						Required:    true,
						Description: "Opsgenie alert priority.",
					},
				},
			},
			"telegram": schema.SingleNestedBlock{
				Description: "Telegram params.",
				Attributes: map[string]schema.Attribute{
					"chat_id": schema.Int64Attribute{
						Required:    true,
						Description: "Telegram chat ID.",
					},
				},
			},
			"teams": schema.SingleNestedBlock{
				Description: "Microsoft Teams params.",
				Attributes: map[string]schema.Attribute{
					"webhook_url": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "Teams webhook URL.",
					},
				},
			},
			"pushover": schema.SingleNestedBlock{
				Description: "Pushover params.",
				Attributes: map[string]schema.Attribute{
					"token": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "Pushover application token.",
					},
					"user_key": schema.StringAttribute{
						Required:    true,
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
						Required:    true,
						Sensitive:   true,
						Description: "Webhook URL.",
					},
				},
			},
			"alertmanager": schema.SingleNestedBlock{
				Description: "Alertmanager params.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Required:    true,
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
						Required:    true,
						Description: "Alert Events V2 endpoint URL.",
					},
					"api_key": schema.StringAttribute{
						Required:    true,
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

	// MonitorIDs.
	if len(ch.MonitorIds) > 0 {
		vals := make([]types.String, len(ch.MonitorIds))
		for i, id := range ch.MonitorIds {
			vals[i] = types.StringValue(strconv.FormatInt(id, 10))
		}
		var d diag.Diagnostics
		m.MonitorIDs, d = types.ListValueFrom(ctx, types.StringType, vals)
		diags.Append(d...)
	} else {
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

	switch ch.Type {
	case generated.Slack:
		p, err := oneOf.AsSlackParams()
		if err != nil {
			warnMalformedParams("slack", err)
			return diags
		}
		sm := &slackModel{}
		if p.AuthMethod != nil {
			sm.AuthMethod = types.StringValue(string(*p.AuthMethod))
		} else {
			sm.AuthMethod = types.StringNull()
		}
		if p.WebhookURL != nil {
			sm.WebhookURL = types.StringValue(*p.WebhookURL)
		} else {
			sm.WebhookURL = types.StringNull()
		}
		if p.Token != nil {
			sm.Token = types.StringValue(*p.Token)
		} else {
			sm.Token = types.StringNull()
		}
		if p.Channel != nil {
			sm.Channel = types.StringValue(*p.Channel)
		} else {
			sm.Channel = types.StringNull()
		}
		m.Slack = sm

	case generated.GoogleChat:
		p, err := oneOf.AsGoogleChatParams()
		if err != nil {
			warnMalformedParams("google_chat", err)
			return diags
		}
		m.GoogleChat = &googleChatModel{
			WebhookURL: types.StringValue(p.WebhookURL),
		}

	case generated.Mattermost:
		p, err := oneOf.AsMattermostParams()
		if err != nil {
			warnMalformedParams("mattermost", err)
			return diags
		}
		m.Mattermost = &mattermostModel{
			WebhookURL: types.StringValue(p.WebhookURL),
		}

	case generated.Pagerduty:
		p, err := oneOf.AsPagerdutyParams()
		if err != nil {
			warnMalformedParams("pagerduty", err)
			return diags
		}
		m.Pagerduty = &pagerdutyModel{
			RoutingKey: types.StringValue(p.RoutingKey),
			Severity:   types.StringValue(string(p.Severity)),
		}

	case generated.Servicenow:
		p, err := oneOf.AsServicenowParams()
		if err != nil {
			warnMalformedParams("servicenow", err)
			return diags
		}
		sm := &servicenowModel{
			URL:      types.StringValue(p.URL),
			Username: types.StringValue(p.Username),
			Password: types.StringValue(p.Password),
		}
		sm.Category = optionalStringToValue(p.Category)
		sm.Subcategory = optionalStringToValue(p.Subcategory)
		if p.Impact != nil {
			sm.Impact = types.StringValue(string(*p.Impact))
		} else {
			sm.Impact = types.StringNull()
		}
		if p.Urgency != nil {
			sm.Urgency = types.StringValue(string(*p.Urgency))
		} else {
			sm.Urgency = types.StringNull()
		}
		if p.Severity != nil {
			sm.Severity = types.StringValue(string(*p.Severity))
		} else {
			sm.Severity = types.StringNull()
		}
		sm.CallerID = optionalStringToValue(p.CallerID)
		sm.Group = optionalStringToValue(p.Group)
		sm.AssignedTo = optionalStringToValue(p.AssignedTo)
		sm.OpenedBy = optionalStringToValue(p.OpenedBy)
		if p.Notify != nil {
			sm.Notify = types.StringValue(string(*p.Notify))
		} else {
			sm.Notify = types.StringNull()
		}
		sm.DueDate = optionalStringToValue(p.DueDate)
		m.Servicenow = sm

	case generated.Opsgenie:
		p, err := oneOf.AsOpsgenieParams()
		if err != nil {
			warnMalformedParams("opsgenie", err)
			return diags
		}
		m.Opsgenie = &opsgenieModel{
			APIKey:   types.StringValue(p.APIKey),
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
		m.Teams = &teamsModel{
			WebhookURL: types.StringValue(p.WebhookURL),
		}

	case generated.Pushover:
		p, err := oneOf.AsPushoverParams()
		if err != nil {
			warnMalformedParams("pushover", err)
			return diags
		}
		pm := &pushoverModel{
			Token:   types.StringValue(p.Token),
			UserKey: types.StringValue(p.UserKey),
		}
		if p.Priority != nil {
			pm.Priority = types.Int64Value(int64(*p.Priority))
		} else {
			pm.Priority = types.Int64Null()
		}
		if p.Sound != nil {
			pm.Sound = types.StringValue(*p.Sound)
		} else {
			pm.Sound = types.StringNull()
		}
		m.Pushover = pm

	case generated.NotificationChannelTypeWebhook:
		p, err := oneOf.AsWebhookParams()
		if err != nil {
			warnMalformedParams("webhook", err)
			return diags
		}
		m.Webhook = &webhookModel{
			URL: types.StringValue(p.URL),
		}

	case generated.Alertmanager:
		p, err := oneOf.AsAlertmanagerParams()
		if err != nil {
			warnMalformedParams("alertmanager", err)
			return diags
		}
		am := &alertmanagerModel{
			URL: types.StringValue(p.URL),
		}
		if p.AuthMethod != nil {
			am.AuthMethod = types.StringValue(string(*p.AuthMethod))
		} else {
			am.AuthMethod = types.StringNull()
		}
		am.Username = optionalStringToValue(p.Username)
		am.Password = optionalStringToValue(p.Password)
		am.Token = optionalStringToValue(p.Token)
		m.Alertmanager = am

	case generated.Incidentio:
		p, err := oneOf.AsIncidentioParams()
		if err != nil {
			warnMalformedParams("incidentio", err)
			return diags
		}
		m.Incidentio = &incidentioModel{
			URL:    types.StringValue(p.URL),
			APIKey: types.StringValue(p.APIKey),
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
