package provider

import (
	"context"
	"crypto/tls"
	"os"

	"github.com/dreamfactorysoftware/df-terraform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure DreamFactoryProvider satisfies various provider interfaces.
var _ provider.Provider = &DreamFactoryProvider{}

// DreamFactoryProvider defines the provider implementation.
type DreamFactoryProvider struct {
	version string
}

// DreamFactoryProviderModel describes the provider data model.
type DreamFactoryProviderModel struct {
	Host               types.String `tfsdk:"host"`
	Email              types.String `tfsdk:"email"`
	Password           types.String `tfsdk:"password"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
}

// New creates a new provider factory function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DreamFactoryProvider{
			version: version,
		}
	}
}

func (p *DreamFactoryProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "dreamfactory"
	resp.Version = p.version
}

func (p *DreamFactoryProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing DreamFactory resources including services, roles, and apps (API keys).",
		MarkdownDescription: `
The DreamFactory provider allows you to manage DreamFactory resources using Terraform.

## Authentication

The provider requires admin credentials to authenticate with DreamFactory. You can provide credentials either through the provider configuration or environment variables.

### Environment Variables

- ` + "`DREAMFACTORY_HOST`" + ` - The URL of the DreamFactory instance
- ` + "`DREAMFACTORY_EMAIL`" + ` - Admin user email
- ` + "`DREAMFACTORY_PASSWORD`" + ` - Admin user password

### Example Usage

` + "```hcl" + `
provider "dreamfactory" {
  host     = "https://api.example.com"
  email    = "admin@example.com"
  password = var.dreamfactory_password
}
` + "```" + `

## Security Recommendations

- Store passwords in environment variables or use a secrets manager
- Use HTTPS in production environments
- Only set ` + "`insecure_skip_verify = true`" + ` for development/testing
`,
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "The URL of the DreamFactory instance (e.g., https://api.example.com). Can be set via DREAMFACTORY_HOST environment variable.",
				Optional:    true,
				Validators: []validator.String{
					&urlValidator{},
				},
			},
			"email": schema.StringAttribute{
				Description: "The email address of a DreamFactory admin user. Can be set via DREAMFACTORY_EMAIL environment variable.",
				Optional:    true,
				Validators: []validator.String{
					&emailValidator{},
				},
			},
			"password": schema.StringAttribute{
				Description: "The password for the DreamFactory admin user. Can be set via DREAMFACTORY_PASSWORD environment variable. It is recommended to use environment variables or a secrets manager for this value.",
				Optional:    true,
				Sensitive:   true,
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Description: "Skip TLS certificate verification. Only use this for development/testing with self-signed certificates. Default: false.",
				Optional:    true,
			},
		},
	}
}

func (p *DreamFactoryProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring DreamFactory provider")

	var config DreamFactoryProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check environment variables for values not provided in config
	host := os.Getenv("DREAMFACTORY_HOST")
	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	email := os.Getenv("DREAMFACTORY_EMAIL")
	if !config.Email.IsNull() {
		email = config.Email.ValueString()
	}

	password := os.Getenv("DREAMFACTORY_PASSWORD")
	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	// Validate required fields
	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing DreamFactory Host",
			"The provider cannot create the DreamFactory API client as there is no host configured. "+
				"Either set it in the provider configuration or use the DREAMFACTORY_HOST environment variable.",
		)
	}

	if email == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("email"),
			"Missing DreamFactory Email",
			"The provider cannot authenticate with DreamFactory as there is no email configured. "+
				"Either set it in the provider configuration or use the DREAMFACTORY_EMAIL environment variable.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing DreamFactory Password",
			"The provider cannot authenticate with DreamFactory as there is no password configured. "+
				"Either set it in the provider configuration or use the DREAMFACTORY_PASSWORD environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Configure TLS settings
	insecureSkipVerify := false
	if !config.InsecureSkipVerify.IsNull() {
		insecureSkipVerify = config.InsecureSkipVerify.ValueBool()
	}

	if insecureSkipVerify {
		tflog.Warn(ctx, "TLS certificate verification is disabled - this should only be used for development/testing")
	}

	// Create a new DreamFactory client with configuration
	clientConfig := client.ClientConfig{
		BaseURL:            host,
		InsecureSkipVerify: insecureSkipVerify,
		TLSMinVersion:      tls.VersionTLS12,
	}
	dfClient := client.NewClientWithConfig(clientConfig)

	tflog.Debug(ctx, "Authenticating with DreamFactory", map[string]interface{}{
		"host":  host,
		"email": email,
	})

	// Login to get a session token
	err := dfClient.LoginWithContext(ctx, email, password)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Authenticate with DreamFactory",
			"An error occurred while attempting to login to DreamFactory. "+
				"Please verify your credentials and ensure the DreamFactory instance is accessible.\n\n"+
				"Error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Successfully authenticated with DreamFactory")

	// Make the client available to resources and data sources
	resp.DataSourceData = dfClient
	resp.ResourceData = dfClient
}

func (p *DreamFactoryProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewServiceResource,
		NewRoleResource,
		NewAppResource,
	}
}

func (p *DreamFactoryProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewServicesDataSource,
	}
}
