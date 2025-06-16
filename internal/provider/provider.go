package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	// NOTE: Change this import path to match your Go module name
	"terraform-provider-flashblade/internal/client"
)

var _ provider.Provider = &flashbladeProvider{}

type flashbladeProvider struct{}

type flashbladeProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	ApiToken types.String `tfsdk:"api_token"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

func New() provider.Provider {
	return &flashbladeProvider{}
}

func (p *flashbladeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "flashblade"
}

func (p *flashbladeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for Pure Storage FlashBlade.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Description: "The API token for the FlashBlade. Can also be set with the FLASHBLADE_API_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"endpoint": schema.StringAttribute{
				Description: "The management VIP or FQDN of the FlashBlade. Can also be set with the FLASHBLADE_ENDPOINT environment variable.",
				Optional:    true,
			},
			"insecure": schema.BoolAttribute{
				Description: "If `true`, the provider will skip TLS certificate verification. This is useful for labs or environments with self-signed certificates, but is not recommended for production. Can also be set with the FLASHBLADE_INSECURE environment variable.",
				Optional:    true,
			},
		},
	}
}

func (p *flashbladeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config flashbladeProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("FLASHBLADE_ENDPOINT")
	apiToken := os.Getenv("FLASHBLADE_API_TOKEN")
	insecure := os.Getenv("FLASHBLADE_INSECURE") == "true"

	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}
	if !config.ApiToken.IsNull() {
		apiToken = config.ApiToken.ValueString()
	}
	if !config.Insecure.IsNull() {
		insecure = config.Insecure.ValueBool()
	}

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(path.Root("endpoint"), "Missing FlashBlade Endpoint", "Set the endpoint in the provider configuration or use the FLASHBLADE_ENDPOINT env var.")
	}
	if apiToken == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_token"), "Missing FlashBlade API Token", "Set the api_token in the provider configuration or use the FLASHBLADE_API_TOKEN env var.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	fbClient, err := client.New(endpoint, apiToken, insecure)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create FlashBlade API Client", "Error: "+err.Error())
		return
	}

	resp.ResourceData = fbClient
	resp.DataSourceData = fbClient
	tflog.Info(ctx, "Configured FlashBlade client", map[string]any{"success": true})
}

func (p *flashbladeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewFileSystemResource, // This registers our file system resource
	}
}

func (p *flashbladeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
