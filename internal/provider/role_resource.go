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
var _ resource.Resource = &RoleResource{}
var _ resource.ResourceWithImportState = &RoleResource{}

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

// RoleResource defines the resource implementation.
type RoleResource struct {
	client *client.Client
}

// RoleResourceModel describes the resource data model.
type RoleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsActive    types.Bool   `tfsdk:"is_active"`
}

func (r *RoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DreamFactory role for access control.",
		MarkdownDescription: `
Manages a DreamFactory role. Roles control what APIs and data users can access.

## Example Usage

` + "```hcl" + `
resource "dreamfactory_role" "api_user" {
  name        = "api_user"
  description = "Standard API user role"
  is_active   = true
}

resource "dreamfactory_role" "admin" {
  name        = "admin"
  description = "Administrator role with full access"
  is_active   = true
}
` + "```" + `

## Assigning Roles to Apps

Roles are typically assigned to apps (API keys) to control access:

` + "```hcl" + `
resource "dreamfactory_app" "mobile_app" {
  name        = "mobile_app"
  description = "Mobile application API key"
  role_id     = dreamfactory_role.api_user.id
}
` + "```" + `

~> **Note:** Role permissions (which services/tables a role can access) must currently be configured in the DreamFactory UI after creating the role.
`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the role.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The unique name of the role. Must start with a letter and contain only letters, numbers, underscores, and hyphens.",
				Required:    true,
				Validators: []validator.String{
					&identifierValidator{},
				},
			},
			"description": schema.StringAttribute{
				Description: "Description of the role.",
				Optional:    true,
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether the role is active. Default: true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func (r *RoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	role := client.Role{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		IsActive:    data.IsActive.ValueBool(),
	}

	tflog.Debug(ctx, "Creating DreamFactory role", map[string]interface{}{
		"name": role.Name,
	})

	id, err := r.client.CreateRole(role)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Create Role",
			fmt.Sprintf("Unable to create role %q: %s\n\n"+
				"Please verify that the role name is unique.",
				data.Name.ValueString(), err),
		)
		return
	}

	data.ID = types.StringValue(strconv.Itoa(id))

	tflog.Info(ctx, "Created DreamFactory role", map[string]interface{}{
		"id":   id,
		"name": data.Name.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Role ID",
			fmt.Sprintf("Unable to parse role ID %q: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Debug(ctx, "Reading DreamFactory role", map[string]interface{}{
		"id": id,
	})

	role, err := r.client.GetRole(id)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			tflog.Info(ctx, "Role not found, removing from state", map[string]interface{}{
				"id": id,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Failed to Read Role",
			fmt.Sprintf("Unable to read role with ID %d: %s", id, err),
		)
		return
	}

	data.Name = types.StringValue(role.Name)
	data.Description = types.StringValue(role.Description)
	data.IsActive = types.BoolValue(role.IsActive)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Role ID",
			fmt.Sprintf("Unable to parse role ID %q: %s", data.ID.ValueString(), err),
		)
		return
	}

	role := client.Role{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		IsActive:    data.IsActive.ValueBool(),
	}

	tflog.Debug(ctx, "Updating DreamFactory role", map[string]interface{}{
		"id":   id,
		"name": role.Name,
	})

	err = r.client.UpdateRole(id, role)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Update Role",
			fmt.Sprintf("Unable to update role %q (ID: %d): %s", data.Name.ValueString(), id, err),
		)
		return
	}

	tflog.Info(ctx, "Updated DreamFactory role", map[string]interface{}{
		"id":   id,
		"name": data.Name.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Role ID",
			fmt.Sprintf("Unable to parse role ID %q: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Debug(ctx, "Deleting DreamFactory role", map[string]interface{}{
		"id":   id,
		"name": data.Name.ValueString(),
	})

	err = r.client.DeleteRole(id)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			// Already deleted, this is fine
			tflog.Info(ctx, "Role already deleted", map[string]interface{}{
				"id": id,
			})
			return
		}
		resp.Diagnostics.AddError(
			"Failed to Delete Role",
			fmt.Sprintf("Unable to delete role %q (ID: %d): %s", data.Name.ValueString(), id, err),
		)
		return
	}

	tflog.Info(ctx, "Deleted DreamFactory role", map[string]interface{}{
		"id":   id,
		"name": data.Name.ValueString(),
	})
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
