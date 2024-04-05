package provider

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/memes/f5xc"
)

// Ensure f5XCProvider satisfies various provider interfaces.
var _ provider.Provider = &f5XCProvider{}

// f5XCProvider defines the provider implementation.
type f5XCProvider struct {
	version string
}

type f5XCConfig struct {
	client  *http.Client
	timeout time.Duration
}

type f5XCProviderModel struct {
	PKCS12File types.String `tfsdk:"api_p12_file"`
	Cert       types.String `tfsdk:"api_cert"`
	Key        types.String `tfsdk:"api_key"`
	Token      types.String `tfsdk:"api_token"`
	Timeout    types.String `tfsdk:"timeout"`
	URL        types.String `tfsdk:"url"`
}

// Create a new provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &f5XCProvider{
			version: version,
		}
	}
}

func (p *f5XCProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "f5xc"
	resp.Version = p.version
}

func (p *f5XCProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_p12_file": schema.StringAttribute{
				Optional: true,
			},
			"api_cert": schema.StringAttribute{
				Optional: true,
			},
			"api_key": schema.StringAttribute{
				Optional: true,
			},
			"api_token": schema.StringAttribute{
				Optional: true,
			},
			"timeout": schema.StringAttribute{
				Optional: true,
			},
			"url": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (p *f5XCProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring F5XC API client")
	var config f5XCProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.PKCS12File.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_p12_file"),
			"Unknown F5XC API PKCS#12 File",
			"The provider cannot create the F5XC API client as there is an unknown configuration value for the F5XC API PKCS#12 file. Either target apply the source of the value first, set the value statically in the configuration, or use the VOLT_API_P12_FILE environment variable.",
		)
	}

	if config.Cert.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_cert"),
			"Unknown F5XC API x509 Certificate File",
			"The provider cannot create the F5XC API client as there is an unknown configuration value for the F5XC API certificate file. Either target apply the source of the value first, set the value statically in the configuration, or use the VOLT_API_CERT environment variable.",
		)
	}

	if config.Key.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown F5XC API x509 Key File",
			"The provider cannot create the F5XC API client as there is an unknown configuration value for the F5XC API key file. Either target apply the source of the value first, set the value statically in the configuration, or use the VOLT_API_KEY environment variable.",
		)
	}

	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Unknown F5XC API auth token",
			"The provider cannot create the F5XC API client as there is an unknown configuration value for the F5XC API authentication token. Either target apply the source of the value first, set the value statically in the configuration, or use the VOLTERRA_TOKEN environment variable.",
		)
	}

	if config.Timeout.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("timeout"),
			"Unknown F5XC API Timeout",
			"The provider cannot create the F5XC API client as there is an unknown configuration value for the F5XC API timeout. Either target apply the source of the value first, set the value statically in the configuration, or use the VOLT_API_TIMEOUT environment variable.",
		)
	}

	if config.URL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Unknown F5XC API URL",
			"The provider cannot create the F5XC API client as there is an unknown configuration value for the F5XC API URL. Either target apply the source of the value first, set the value statically in the configuration, or use the VOLT_API_URL environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	apiP12File := os.Getenv("VOLT_API_P12_FILE")
	apiP12Passphrase := os.Getenv("VES_P12_PASSWORD")
	apiCert := os.Getenv("VOLT_API_CERT")
	apiKey := os.Getenv("VOLT_API_KEY")
	apiToken := os.Getenv("VOLTERRA_TOKEN")
	timeoutValue := os.Getenv("VOLT_API_TIMEOUT")
	url := os.Getenv("VOLT_API_URL")

	if !config.PKCS12File.IsNull() {
		apiP12File = config.PKCS12File.ValueString()
	}
	if !config.Cert.IsNull() {
		apiCert = config.Cert.ValueString()
	}
	if !config.Key.IsNull() {
		apiKey = config.Key.ValueString()
	}
	if !config.Token.IsNull() {
		apiToken = config.Token.ValueString()
	}
	if !config.Timeout.IsNull() {
		timeoutValue = config.Timeout.ValueString()
	}
	if !config.URL.IsNull() {
		url = config.URL.ValueString()
	}

	timeout := 20 * time.Second
	if timeoutValue != "" {
		t, err := time.ParseDuration(timeoutValue)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to parse timeout",
				"An unexpected error occurred when parsing API timeout. "+
					"If the error is not clear, please contact the provider developers.\n\n"+
					"F5XC Client Error: "+err.Error(),
			)
			return
		}
		timeout = t
	}

	// url is required to be set
	if url == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Missing F5XC API URL",
			"The provider cannot create the F5XC API client as there is a missing or empty value for the F5XC API URL. "+
				"Set the url value in the configuration or use the VOLT_API_URL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	options := []f5xc.Option{
		f5xc.WithAPIEndpoint(url),
	}
	if apiToken != "" {
		options = append(options, f5xc.WithAuthToken(apiToken))
	}
	if apiP12File != "" && apiP12Passphrase != "" {
		options = append(options, f5xc.WithP12Certificate(apiP12File, apiP12Passphrase))
	}
	if apiCert != "" && apiKey != "" {
		options = append(options, f5xc.WithCertKeyPair(apiCert, apiKey))
	}
	client, err := f5xc.NewClient(options...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create F5XC API Client",
			"An unexpected error occurred when creating the F5XC API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"F5XC Client Error: "+err.Error(),
		)
		return
	}
	cfg := f5XCConfig{
		client:  client,
		timeout: timeout,
	}
	resp.DataSourceData = &cfg
	resp.ResourceData = &cfg
}

func (p *f5XCProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBlindfoldResource,
		NewBlindfoldFileResource,
	}
}

// The provider does not expose any datasources to Terraform.
func (p *f5XCProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
