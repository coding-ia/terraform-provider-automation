package provider

import (
	"context"
	"github.com/coding-ia/terraform-provider-automation/internal/conn"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &AutomationProvider{}

type AutomationProvider struct {
	version string
	Meta    Meta
}

type ProviderConfigurationModel struct {
	Profile types.String `tfsdk:"profile"`
	Region  types.String `tfsdk:"region"`
}

func (ap *AutomationProvider) Metadata(ctx context.Context, request provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "automation"
	response.Version = ap.version
}

func (ap *AutomationProvider) Schema(ctx context.Context, request provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "The automation Terraform provider contains various resources used to assist in automation.",
		Attributes: map[string]schema.Attribute{
			"profile": schema.StringAttribute{
				Description: "The profile for API operations. If not set, the default profile for aws configuration will be used.",
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region in AWS where actions will take place.",
				Optional:    true,
			},
		},
	}
}

func (ap *AutomationProvider) Configure(ctx context.Context, request provider.ConfigureRequest, response *provider.ConfigureResponse) {
	var config ProviderConfigurationModel

	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)

	if response.Diagnostics.HasError() {
		return
	}

	opts := &conn.AWSConfigOptions{
		Profile: config.Profile.ValueString(),
		Region:  config.Region.ValueString(),
	}

	client := conn.CreateAWSClient(ctx, opts)
	ap.Meta.AWSClient = *client

	response.ResourceData = ap.Meta
}

func (ap *AutomationProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (ap *AutomationProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newAWSSSMAssociationResource,
		newAWSSSMStartAutomationResource,
	}
}

func (ap *AutomationProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AutomationProvider{
			version: version,
		}
	}
}
