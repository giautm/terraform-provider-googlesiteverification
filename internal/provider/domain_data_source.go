package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/api/siteverification/v1"
)

type (
	// DomainDataSource defines the data source implementation.
	DomainDataSource struct {
		srv *siteverification.Service
	}
	// DomainDataSourceModel describes the data source data model.
	DomainDataSourceModel struct {
		ID          types.String `tfsdk:"id"`
		RecordType  types.String `tfsdk:"record_type"`
		RecordName  types.String `tfsdk:"record_name"`
		RecordValue types.String `tfsdk:"record_value"`
		Timeouts    types.Object `tfsdk:"timeouts"`
	}
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ datasource.DataSource = &DomainDataSource{}
)

const (
	resourceType       = "INET_DOMAIN"
	verificationMethod = "DNS_TXT"
)

func NewDomainDataSource() datasource.DataSource {
	return &DomainDataSource{}
}

func (d *DomainDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *DomainDataSource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "The Domain data source provides a token for verifying domain ownership.",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				MarkdownDescription: "The domain you want to verify.",
				Type:                types.StringType,
				Required:            true,
			},
			"record_type": {
				MarkdownDescription: "The type of DNS record you should create.",
				Type:                types.StringType,
				Computed:            true,
			},
			"record_name": {
				MarkdownDescription: "The name of the record you should create.",
				Type:                types.StringType,
				Computed:            true,
			},
			"record_value": {
				MarkdownDescription: "The value of the record you should create.",
				Type:                types.StringType,
				Computed:            true,
			},
		},
		Blocks: map[string]tfsdk.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Read: true,
			}),
		},
	}, nil
}

func (d *DomainDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	srv, ok := req.ProviderData.(*siteverification.Service)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *siteverification.Service, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.srv = srv
}

func (d *DomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DomainDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout := timeouts.Read(ctx, data.Timeouts, 60*time.Second)
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	result, err := d.srv.WebResource.
		GetToken(&siteverification.SiteVerificationWebResourceGettokenRequest{
			Site: &siteverification.SiteVerificationWebResourceGettokenRequestSite{
				Identifier: data.ID.Value,
				Type:       resourceType,
			},
			VerificationMethod: verificationMethod,
		}).
		Do()
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to read DNS Token, got error: %s", err),
		)
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	data.RecordType = types.String{Value: "TXT"}
	data.RecordName = data.ID
	data.RecordValue = types.String{Value: result.Token}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
