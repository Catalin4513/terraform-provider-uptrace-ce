package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &OrgResource{}
	_ resource.ResourceWithConfigure   = &OrgResource{}
	_ resource.ResourceWithImportState = &OrgResource{}
)

// NewOrgResource creates the org resource.
func NewOrgResource() resource.Resource {
	return &OrgResource{}
}

// OrgResource manages an Uptrace organization.
type OrgResource struct {
	client *Client
}

// OrgResourceModel describes the Terraform state for an organization.
type OrgResourceModel struct {
	ID        types.String  `tfsdk:"id"`
	Name      types.String  `tfsdk:"name"`
	Budget    types.Float64 `tfsdk:"budget"`
	CreatedAt types.String  `tfsdk:"created_at"`
	UpdatedAt types.String  `tfsdk:"updated_at"`
}

// apiOrg is the JSON shape returned by the Uptrace API.
type apiOrg struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Budget    float64 `json:"budget"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

func (r *OrgResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org"
}

func (r *OrgResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Uptrace organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Organization identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Organization name.",
				Required:    true,
			},
			"budget": schema.Float64Attribute{
				Description: "Organization budget. Can only be set on creation.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Last update timestamp.",
				Computed:    true,
			},
		},
	}
}

func (r *OrgResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", fmt.Sprintf("Expected *Client, got %T", req.ProviderData))
		return
	}
	r.client = client
}

// Create sends POST /internal/v1/orgs.
func (r *OrgResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
	}
	if !plan.Budget.IsNull() && !plan.Budget.IsUnknown() {
		body["budget"] = plan.Budget.ValueFloat64()
	}

	tflog.Info(ctx, "Creating org", map[string]any{"name": plan.Name.ValueString()})

	org, err := r.doOrgRequest(ctx, http.MethodPost, "/orgs", body)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Organization", err.Error())
		return
	}

	orgToState(org, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read sends GET /internal/v1/orgs/{orgId}.
func (r *OrgResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := r.doOrgRequest(ctx, http.MethodGet, "/orgs/"+state.ID.ValueString(), nil)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Organization", err.Error())
		return
	}

	orgToState(org, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update sends PUT /internal/v1/orgs/{orgId}.
func (r *OrgResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrgResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
	}

	tflog.Info(ctx, "Updating org", map[string]any{"id": plan.ID.ValueString()})

	org, err := r.doOrgRequest(ctx, http.MethodPut, "/orgs/"+plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Organization", err.Error())
		return
	}

	orgToState(org, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete sends DELETE /internal/v1/orgs/{orgId}.
func (r *OrgResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting org", map[string]any{"id": state.ID.ValueString()})

	url := r.client.Endpoint + "/orgs/" + state.ID.ValueString()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Organization", err.Error())
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+r.client.Token)

	httpResp, err := r.client.HTTP.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Organization", err.Error())
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		if strings.Contains(string(bodyBytes), "not found") || httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Organization",
			fmt.Sprintf("status %d: %s", httpResp.StatusCode, string(bodyBytes)))
	}
}

func (r *OrgResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// doOrgRequest sends an HTTP request and parses the org from the response.
func (r *OrgResource) doOrgRequest(ctx context.Context, method, urlPath string, body map[string]any) (*apiOrg, error) {
	url := r.client.Endpoint + urlPath

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+r.client.Token)
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	httpResp, err := r.client.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var result struct {
		Org apiOrg `json:"org"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result.Org, nil
}

// orgToState maps API response to Terraform state.
func orgToState(org *apiOrg, state *OrgResourceModel) {
	state.ID = types.StringValue(strconv.FormatInt(org.ID, 10))
	state.Name = types.StringValue(org.Name)

	if org.Budget != 0 {
		state.Budget = types.Float64Value(org.Budget)
	} else {
		state.Budget = types.Float64Null()
	}

	if org.CreatedAt != "" {
		state.CreatedAt = types.StringValue(org.CreatedAt)
	}
	if org.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(org.UpdatedAt)
	}
}
