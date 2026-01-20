package provider

import (
	"context"
	"fmt"

	"github.com/dreamfactorysoftware/df-terraform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider-defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ServicesDataSource{}

func NewServicesDataSource() datasource.DataSource {
	return &ServicesDataSource{}
}

// ServicesDataSource defines the data source implementation.
type ServicesDataSource struct {
	client *client.Client
}

// ServicesDataSourceModel describes the data source data model.
type ServicesDataSourceModel struct {
	Services []ServiceDataModel `tfsdk:"services"`
}

// ServiceDataModel describes a single service in the data source.
type ServiceDataModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Label       types.String `tfsdk:"label"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	IsActive    types.Bool   `tfsdk:"is_active"`
}

func (d *ServicesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_services"
}

func (d *ServicesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of all DreamFactory services.",

		Attributes: map[string]schema.Attribute{
			"services": schema.ListNestedAttribute{
				Description: "List of services.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the service.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The unique name of the service.",
							Computed:    true,
						},
						"label": schema.StringAttribute{
							Description: "Display label for the service.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the service.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of service.",
							Computed:    true,
						},
						"is_active": schema.BoolAttribute{
							Description: "Whether the service is active.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ServicesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *ServicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServicesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	services, err := d.client.ListServices()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list services: %s", err))
		return
	}

	data.Services = make([]ServiceDataModel, len(services))
	for i, svc := range services {
		data.Services[i] = ServiceDataModel{
			ID:          types.StringValue(fmt.Sprintf("%d", svc.ID)),
			Name:        types.StringValue(svc.Name),
			Label:       types.StringValue(svc.Label),
			Description: types.StringValue(svc.Description),
			Type:        types.StringValue(svc.Type),
			IsActive:    types.BoolValue(svc.IsActive),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
