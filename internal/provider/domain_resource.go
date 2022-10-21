package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/api/siteverification/v1"
)

type (
	// DomainResource defines the resource implementation.
	DomainResource struct {
		srv *siteverification.Service
	}
	// DomainResourceModel describes the resource data model.
	DomainResourceModel struct {
		Domain types.String `tfsdk:"domain"`
		Token  types.String `tfsdk:"token"`
		Id     types.String `tfsdk:"id"`
	}
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &DomainResource{}
	_ resource.ResourceWithImportState = &DomainResource{}
)

func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

func (r *DomainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns"
}

func (r *DomainResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Manages a DNS verification for a domain.",
		Attributes: map[string]tfsdk.Attribute{
			"domain": {
				MarkdownDescription: "The domain you want to verify.",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
				},
			},
			"token": {
				MarkdownDescription: "The token you got from data.googlesiteverification_dns_token. This forces a new verification in case the token changes.",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
				},
			},
			"id": {
				Computed:            true,
				MarkdownDescription: "The id of the verification.",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
		},
	}, nil
}

func (r *DomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	srv, ok := req.ProviderData.(*siteverification.Service)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *siteverification.Service, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.srv = srv
}

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *DomainResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.srv.WebResource.
		Insert(verificationMethod, &siteverification.SiteVerificationWebResourceResource{
			Site: &siteverification.SiteVerificationWebResourceResourceSite{
				Identifier: data.Domain.Value,
				Type:       resourceType,
			},
		}).
		Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to create verification, got error: %s", err))
		return
	}

	id, err := url.QueryUnescape(result.Id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("failed to urldecode id %s, %s", result.Id, err))
		return
	}
	data.Id = types.String{Value: id}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *DomainResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.srv.WebResource.Get(data.Id.Value).Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to read verification, got error: %s", err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Do nothing because we have RequiresReplace on domain and token
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *DomainResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.srv.WebResource.Delete(data.Id.Value).Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to delete verification, got error: %s", err))
		return
	}
}

func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	_, err := r.srv.WebResource.Get(req.ID).Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to import verification, got error: %s", err))
		return
	}
	domain := strings.TrimPrefix(req.ID, "dns://")

	result, err := r.srv.WebResource.
		GetToken(&siteverification.SiteVerificationWebResourceGettokenRequest{
			Site: &siteverification.SiteVerificationWebResourceGettokenRequestSite{
				Identifier: domain,
				Type:       resourceType,
			},
			VerificationMethod: verificationMethod,
		}).
		Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to import verification, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &DomainResourceModel{
		Id:     types.String{Value: req.ID},
		Domain: types.String{Value: domain},
		Token:  types.String{Value: result.Token},
	})...)
}
