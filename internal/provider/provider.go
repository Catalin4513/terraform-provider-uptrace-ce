package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Client is a minimal HTTP client for the Uptrace API.
type Client struct {
	Endpoint  string
	Token     string
	ProjectID int64
	HTTP      *http.Client
}

// UptraceProvider implements the Terraform provider interface.
type UptraceProvider struct {
	version string
}

// UptraceProviderModel describes the provider config.
type UptraceProviderModel struct {
	Endpoint  types.String `tfsdk:"endpoint"`
	Token     types.String `tfsdk:"token"`
	ProjectID types.Int64  `tfsdk:"project_id"`
}

// New returns a provider constructor.
func New(version string) func() tfprovider.Provider {
	return func() tfprovider.Provider {
		return &UptraceProvider{version: version}
	}
}

func (p *UptraceProvider) Metadata(_ context.Context, _ tfprovider.MetadataRequest, resp *tfprovider.MetadataResponse) {
	resp.TypeName = "uptrace"
	resp.Version = p.version
}

func (p *UptraceProvider) Schema(_ context.Context, _ tfprovider.SchemaRequest, resp *tfprovider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Minimal Terraform provider for Uptrace.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "Uptrace API endpoint. Can also be set via UPTRACE_ENDPOINT.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "Uptrace API token. Can also be set via UPTRACE_TOKEN.",
				Optional:    true,
				Sensitive:   true,
			},
			"project_id": schema.Int64Attribute{
				Description: "Default project ID. Can also be set via UPTRACE_PROJECT_ID.",
				Optional:    true,
			},
		},
	}
}

func (p *UptraceProvider) Configure(ctx context.Context, req tfprovider.ConfigureRequest, resp *tfprovider.ConfigureResponse) {
	var config UptraceProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("UPTRACE_ENDPOINT")
	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}
	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(path.Root("endpoint"), "Missing endpoint",
			"Set endpoint in config or UPTRACE_ENDPOINT env var.")
		return
	}

	token := os.Getenv("UPTRACE_TOKEN")
	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}
	if token == "" {
		resp.Diagnostics.AddAttributeError(path.Root("token"), "Missing token",
			"Set token in config or UPTRACE_TOKEN env var.")
		return
	}

	var projectID int64
	if !config.ProjectID.IsNull() {
		projectID = config.ProjectID.ValueInt64()
	} else if env := os.Getenv("UPTRACE_PROJECT_ID"); env != "" {
		var err error
		projectID, err = strconv.ParseInt(env, 10, 64)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("project_id"), "Invalid project ID",
				fmt.Sprintf("Cannot parse UPTRACE_PROJECT_ID %q: %s", env, err))
			return
		}
	}

	client := &Client{
		Endpoint:  endpoint,
		Token:     token,
		ProjectID: projectID,
		HTTP:      http.DefaultClient,
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *UptraceProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewOrgResource,
	}
}

func (p *UptraceProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
