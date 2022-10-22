package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/api/siteverification/v1"
)

type (
	// GoogleSiteVerificationProvider defines the provider implementation.
	GoogleSiteVerificationProvider struct {
		// version is set to the provider version on release, "dev" when the
		// provider is built and ran locally, and "test" when running acceptance
		// testing.
		version string
	}
	// GoogleSiteVerificationProviderModel describes the provider data model.
	GoogleSiteVerificationProviderModel struct {
		Credentials types.String `tfsdk:"credentials"`
	}
)

// Ensure GoogleSiteVerificationProvider satisfies various provider interfaces.
var (
	_ provider.Provider             = &GoogleSiteVerificationProvider{}
	_ provider.ProviderWithMetadata = &GoogleSiteVerificationProvider{}
)

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GoogleSiteVerificationProvider{
			version: version,
		}
	}
}

func (p *GoogleSiteVerificationProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "googlesiteverification"
	resp.Version = p.version
}

func (p *GoogleSiteVerificationProvider) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"credentials": {
				MarkdownDescription: "Either the path to or the contents of a [service account key file](https://cloud.google.com/iam/docs/creating-managing-service-account-keys) in JSON format. If not provided, the [application default credentials](https://cloud.google.com/sdk/gcloud/reference/auth/application-default) will be used.",
				Optional:            true,
				Type:                types.StringType,
			},
		},
	}, nil
}

func (p *GoogleSiteVerificationProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data GoogleSiteVerificationProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	// if data.Endpoint.IsNull() { /* ... */ }
	srv, err := siteverification.NewService(context.Background())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create siteverification service",
			fmt.Sprintf("Unable to create siteverification service: %s", err),
		)
	}
	resp.DataSourceData = srv
	resp.ResourceData = srv
}

func (p *GoogleSiteVerificationProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDomainResource,
	}
}

func (p *GoogleSiteVerificationProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDomainDataSource,
	}
}
