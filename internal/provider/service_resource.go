package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/dreamfactorysoftware/df-terraform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider-defined types fully satisfy framework interfaces.
var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithImportState = &ServiceResource{}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

// ServiceResource defines the resource implementation.
type ServiceResource struct {
	client *client.Client
}

// ServiceResourceModel describes the resource data model.
type ServiceResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Label       types.String `tfsdk:"label"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	IsActive    types.Bool   `tfsdk:"is_active"`

	// Database connection config (common fields)
	Host     types.String `tfsdk:"host"`
	Port     types.Int64  `tfsdk:"port"`
	Database types.String `tfsdk:"database"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Schema   types.String `tfsdk:"schema"`
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DreamFactory service (API endpoint).",
		MarkdownDescription: `
Manages a DreamFactory service. Services are the core building blocks in DreamFactory that expose your data sources as REST APIs.

## Example Usage

### MySQL Database Service

` + "```hcl" + `
resource "dreamfactory_service" "mysql_db" {
  name        = "customer_db"
  label       = "Customer Database"
  description = "Customer data from MySQL"
  type        = "mysql"
  is_active   = true

  host     = "mysql.example.com"
  port     = 3306
  database = "customers"
  username = "api_user"
  password = var.mysql_password
}
` + "```" + `

### PostgreSQL Database Service

` + "```hcl" + `
resource "dreamfactory_service" "postgres_db" {
  name        = "orders_db"
  label       = "Orders Database"
  type        = "pgsql"

  host     = "postgres.example.com"
  port     = 5432
  database = "orders"
  username = "api_user"
  password = var.postgres_password
  schema   = "public"
}
` + "```" + `
`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the service.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The unique name of the service (used in API URLs). Must start with a letter and contain only letters, numbers, underscores, and hyphens.",
				Required:    true,
				Validators: []validator.String{
					&identifierValidator{},
				},
			},
			"label": schema.StringAttribute{
				Description: "Display label for the service.",
				Optional:    true,
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the service.",
				Optional:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of service (e.g., mysql, pgsql, mongodb, snowflake, sqlsrv, oracle).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					&serviceTypeValidator{},
				},
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether the service is active. Default: true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			// Database connection attributes
			"host": schema.StringAttribute{
				Description: "Database host address.",
				Optional:    true,
			},
			"port": schema.Int64Attribute{
				Description: "Database port (1-65535).",
				Optional:    true,
				Validators: []validator.Int64{
					&portValidator{},
				},
			},
			"database": schema.StringAttribute{
				Description: "Database name.",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "Database username.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Database password. This value is sensitive and will not be displayed in logs or state.",
				Optional:    true,
				Sensitive:   true,
			},
			"schema": schema.StringAttribute{
				Description: "Database schema (for PostgreSQL, SQL Server, etc.).",
				Optional:    true,
			},
		},
	}
}

func (r *ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the service config - only include fields that are set and not unknown
	config := make(client.ServiceConfig)
	if !data.Host.IsNull() && !data.Host.IsUnknown() {
		config["host"] = data.Host.ValueString()
	}
	if !data.Port.IsNull() && !data.Port.IsUnknown() {
		config["port"] = int(data.Port.ValueInt64())
	}
	if !data.Database.IsNull() && !data.Database.IsUnknown() {
		config["database"] = data.Database.ValueString()
	}
	if !data.Username.IsNull() && !data.Username.IsUnknown() {
		config["username"] = data.Username.ValueString()
	}
	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		config["password"] = data.Password.ValueString()
	}
	if !data.Schema.IsNull() && !data.Schema.IsUnknown() {
		config["schema"] = data.Schema.ValueString()
	}

	// Set label to name if not provided
	label := data.Label.ValueString()
	if data.Label.IsNull() || data.Label.IsUnknown() {
		label = data.Name.ValueString()
	}

	service := client.Service{
		Name:        data.Name.ValueString(),
		Label:       label,
		Description: data.Description.ValueString(),
		Type:        data.Type.ValueString(),
		IsActive:    data.IsActive.ValueBool(),
		Config:      config,
	}

	tflog.Debug(ctx, "Creating DreamFactory service", map[string]interface{}{
		"name": service.Name,
		"type": service.Type,
	})

	id, err := r.client.CreateService(service)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Create Service",
			fmt.Sprintf("Unable to create service %q of type %q: %s\n\n"+
				"Please verify that:\n"+
				"- The service name is unique\n"+
				"- The service type is valid\n"+
				"- Database connection details are correct (if applicable)",
				data.Name.ValueString(), data.Type.ValueString(), err),
		)
		return
	}

	data.ID = types.StringValue(strconv.Itoa(id))

	// Set label in state
	if data.Label.IsNull() || data.Label.IsUnknown() {
		data.Label = data.Name
	}

	tflog.Info(ctx, "Created DreamFactory service", map[string]interface{}{
		"id":   id,
		"name": data.Name.ValueString(),
		"type": data.Type.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Service ID",
			fmt.Sprintf("Unable to parse service ID %q: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Debug(ctx, "Reading DreamFactory service", map[string]interface{}{
		"id": id,
	})

	service, err := r.client.GetService(id)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			tflog.Info(ctx, "Service not found, removing from state", map[string]interface{}{
				"id": id,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Failed to Read Service",
			fmt.Sprintf("Unable to read service with ID %d: %s", id, err),
		)
		return
	}

	// Preserve password from state (API returns masked password)
	password := data.Password

	data.Name = types.StringValue(service.Name)
	data.Label = types.StringValue(service.Label)
	data.Description = types.StringValue(service.Description)
	data.Type = types.StringValue(service.Type)
	data.IsActive = types.BoolValue(service.IsActive)

	// Read config values
	if service.Config != nil {
		if host, ok := service.Config["host"].(string); ok {
			data.Host = types.StringValue(host)
		} else {
			data.Host = types.StringNull()
		}
		if port, ok := service.Config["port"].(float64); ok {
			data.Port = types.Int64Value(int64(port))
		} else {
			data.Port = types.Int64Null()
		}
		if database, ok := service.Config["database"].(string); ok {
			data.Database = types.StringValue(database)
		} else {
			data.Database = types.StringNull()
		}
		if username, ok := service.Config["username"].(string); ok {
			data.Username = types.StringValue(username)
		} else {
			data.Username = types.StringNull()
		}
		if schema, ok := service.Config["schema"].(string); ok {
			data.Schema = types.StringValue(schema)
		} else {
			data.Schema = types.StringNull()
		}
	}

	// Restore password from state (API returns masked value)
	data.Password = password

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Service ID",
			fmt.Sprintf("Unable to parse service ID %q: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Build the service config - only include fields that are set and not unknown
	config := make(client.ServiceConfig)
	if !data.Host.IsNull() && !data.Host.IsUnknown() {
		config["host"] = data.Host.ValueString()
	}
	if !data.Port.IsNull() && !data.Port.IsUnknown() {
		config["port"] = int(data.Port.ValueInt64())
	}
	if !data.Database.IsNull() && !data.Database.IsUnknown() {
		config["database"] = data.Database.ValueString()
	}
	if !data.Username.IsNull() && !data.Username.IsUnknown() {
		config["username"] = data.Username.ValueString()
	}
	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		config["password"] = data.Password.ValueString()
	}
	if !data.Schema.IsNull() && !data.Schema.IsUnknown() {
		config["schema"] = data.Schema.ValueString()
	}

	service := client.Service{
		Name:        data.Name.ValueString(),
		Label:       data.Label.ValueString(),
		Description: data.Description.ValueString(),
		Type:        data.Type.ValueString(),
		IsActive:    data.IsActive.ValueBool(),
		Config:      config,
	}

	tflog.Debug(ctx, "Updating DreamFactory service", map[string]interface{}{
		"id":   id,
		"name": service.Name,
	})

	err = r.client.UpdateService(id, service)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Update Service",
			fmt.Sprintf("Unable to update service %q (ID: %d): %s", data.Name.ValueString(), id, err),
		)
		return
	}

	tflog.Info(ctx, "Updated DreamFactory service", map[string]interface{}{
		"id":   id,
		"name": data.Name.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Service ID",
			fmt.Sprintf("Unable to parse service ID %q: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Debug(ctx, "Deleting DreamFactory service", map[string]interface{}{
		"id":   id,
		"name": data.Name.ValueString(),
	})

	err = r.client.DeleteService(id)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			// Already deleted, this is fine
			tflog.Info(ctx, "Service already deleted", map[string]interface{}{
				"id": id,
			})
			return
		}
		resp.Diagnostics.AddError(
			"Failed to Delete Service",
			fmt.Sprintf("Unable to delete service %q (ID: %d): %s", data.Name.ValueString(), id, err),
		)
		return
	}

	tflog.Info(ctx, "Deleted DreamFactory service", map[string]interface{}{
		"id":   id,
		"name": data.Name.ValueString(),
	})
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
