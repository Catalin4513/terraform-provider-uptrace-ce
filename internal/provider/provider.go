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

// UptraceProvider implements the Terraform provider for Uptrace.
type UptraceProvider struct {
	version string
}

// uptraceProviderModel maps the provider HCL config to Go types.
type uptraceProviderModel struct {
	Endpoint  types.String `tfsdk:"endpoint"`
	Token     types.String `tfsdk:"token"`
	ProjectID types.Int64  `tfsdk:"project_id"`
}

// New returns a factory function that creates the provider.
func New(version string) func() tfprovider.Provider {
	return func() tfprovider.Provider {
		return &UptraceProvider{version: version}
	}
}

// Metadata sets the provider type name.
func (p *UptraceProvider) Metadata(_ context.Context, _ tfprovider.MetadataRequest, resp *tfprovider.MetadataResponse) {
	resp.TypeName = "uptrace"
	resp.Version = p.version
}

// Schema defines the provider configuration attributes.
func (p *UptraceProvider) Schema(_ context.Context, _ tfprovider.SchemaRequest, resp *tfprovider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional: true,
			},
			"token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"project_id": schema.Int64Attribute{
				Optional: true,
			},
		},
	}
}

// Configure creates the API client from provider config and env vars.
func (p *UptraceProvider) Configure(ctx context.Context, req tfprovider.ConfigureRequest, resp *tfprovider.ConfigureResponse) {
	var conf uptraceProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &conf)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := envOrConfig("UPTRACE_ENDPOINT", conf.Endpoint)
	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"), "Missing endpoint",
			"Set endpoint in config or UPTRACE_ENDPOINT env var.")
		return
	}

	token := envOrConfig("UPTRACE_TOKEN", conf.Token)
	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"), "Missing token",
			"Set token in config or UPTRACE_TOKEN env var.")
		return
	}

	projectID, err := parseProjectID(conf.ProjectID)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("project_id"), "Invalid project ID", err.Error())
		return
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

// Resources returns the list of managed resource types.
func (p *UptraceProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewOrgResource,
	}
}

// DataSources returns the list of data source types.
func (p *UptraceProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// envOrConfig returns the config value if set, otherwise falls back to the env var.
func envOrConfig(envKey string, configVal types.String) string {
	if !configVal.IsNull() {
		return configVal.ValueString()
	}
	return os.Getenv(envKey)
}

// parseProjectID extracts the project ID from config or env var.
func parseProjectID(configVal types.Int64) (int64, error) {
	if !configVal.IsNull() {
		return configVal.ValueInt64(), nil
	}

	env := os.Getenv("UPTRACE_PROJECT_ID")
	if env == "" {
		return 0, nil
	}

	id, err := strconv.ParseInt(env, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse UPTRACE_PROJECT_ID %q: %w", env, err)
	}
	return id, nil
}
