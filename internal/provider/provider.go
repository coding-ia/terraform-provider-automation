package provider

import (
	"context"
	"github.com/coding-ia/terraform-provider-automation/internal/conn"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &AutomationProvider{}

type AutomationProvider struct {
	version string
	Meta    Meta
}

func (ap *AutomationProvider) Metadata(ctx context.Context, request provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "automation"
	response.Version = ap.version
}

func (ap *AutomationProvider) Schema(ctx context.Context, request provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "The automation Terraform provider contains various resources used to assist in automation.",
	}
}

func (ap *AutomationProvider) Configure(ctx context.Context, request provider.ConfigureRequest, response *provider.ConfigureResponse) {
	client := conn.CreateAWSClient(ctx)
	ap.Meta.AWSClient = *client

	response.ResourceData = ap.Meta
}

func (ap *AutomationProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (ap *AutomationProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newAWSSSMAssociationResource,
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
