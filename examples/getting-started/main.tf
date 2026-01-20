# =============================================================================
# DreamFactory Terraform Provider - Getting Started Guide
# =============================================================================
#
# This example demonstrates how to use Terraform to manage DreamFactory resources.
#
# WHAT IS TERRAFORM?
# Terraform is an Infrastructure as Code (IaC) tool. Instead of clicking around
# in a UI to create resources, you define what you want in code files (.tf files).
# Terraform then creates/updates/deletes resources to match your definition.
#
# KEY CONCEPTS:
# - "resource" = something Terraform creates and manages (a service, role, app)
# - "data source" = read-only query to fetch existing data
# - "output" = values displayed after terraform apply (like API keys)
# - "variable" = input parameters you can change without editing the main code
#
# =============================================================================

# -----------------------------------------------------------------------------
# STEP 1: Configure Terraform and the Provider
# -----------------------------------------------------------------------------
# This block tells Terraform which provider to use and where to get it.

terraform {
  required_providers {
    dreamfactory = {
      source  = "registry.terraform.io/dreamfactorysoftware/dreamfactory"
      version = "0.1.0"
    }
  }
}

# -----------------------------------------------------------------------------
# STEP 2: Configure the DreamFactory Provider
# -----------------------------------------------------------------------------
# This tells the provider how to connect to your DreamFactory instance.
# You need admin credentials to manage resources.

provider "dreamfactory" {
  host     = var.dreamfactory_host      # URL of your DreamFactory instance
  email    = var.dreamfactory_email     # Admin email
  password = var.dreamfactory_password  # Admin password (sensitive!)
}

# -----------------------------------------------------------------------------
# STEP 3: Define Input Variables
# -----------------------------------------------------------------------------
# Variables let you customize the configuration without editing this file.
# You can set these via:
#   - terraform.tfvars file
#   - Environment variables (TF_VAR_name)
#   - Command line: terraform apply -var="name=value"

variable "dreamfactory_host" {
  type        = string
  description = "URL of the DreamFactory instance"
  default     = "http://localhost"
}

variable "dreamfactory_email" {
  type        = string
  description = "Admin email for DreamFactory"
  default     = "admin@example.com"
}

variable "dreamfactory_password" {
  type        = string
  sensitive   = true  # Won't be shown in logs
  description = "Admin password for DreamFactory"
}

# Database connection variables
variable "db_host" {
  type        = string
  description = "Database server hostname"
  default     = "example_data"  # Docker container name from docker-compose
}

variable "db_password" {
  type        = string
  sensitive   = true
  description = "Database password"
  default     = "root_pw"
}

# -----------------------------------------------------------------------------
# STEP 4: Create a Role
# -----------------------------------------------------------------------------
# Roles define what API users can access. You assign a role to an app,
# and anyone using that app's API key gets those permissions.

resource "dreamfactory_role" "api_readonly" {
  name        = "api_readonly"
  description = "Read-only access to customer data"
  is_active   = true
}

resource "dreamfactory_role" "api_readwrite" {
  name        = "api_readwrite"
  description = "Full read/write access to customer data"
  is_active   = true
}

# -----------------------------------------------------------------------------
# STEP 5: Create Database Services (API Endpoints)
# -----------------------------------------------------------------------------
# Services expose databases (or other backends) as REST APIs.
# Once created, you can access data at /api/v2/{service_name}/_table/{table}

resource "dreamfactory_service" "customer_db" {
  name        = "customers"
  label       = "Customer Database"
  description = "PostgreSQL database with customer information"
  type        = "pgsql"  # PostgreSQL. Use "mysql" for MySQL, "sqlsrv" for SQL Server
  is_active   = true

  # Database connection settings
  host     = var.db_host
  port     = 5432
  database = "pagila"      # Sample database in the example_data container
  username = "postgres"
  password = var.db_password
}

# -----------------------------------------------------------------------------
# STEP 6: Create Apps (API Keys)
# -----------------------------------------------------------------------------
# Apps generate API keys. External systems use these keys to authenticate.
# You can assign a role to control what the API key can access.

resource "dreamfactory_app" "mobile_app" {
  name        = "mobile_app"
  description = "API key for mobile application - read only access"
  is_active   = true
  role_id     = dreamfactory_role.api_readonly.id  # Uses output from the role resource!
}

resource "dreamfactory_app" "backend_service" {
  name        = "backend_service"
  description = "API key for backend service - full access"
  is_active   = true
  role_id     = dreamfactory_role.api_readwrite.id
}

resource "dreamfactory_app" "public_api" {
  name        = "public_api"
  description = "API key for public access - no role restrictions"
  is_active   = true
  # No role_id means admin access - be careful!
}

# -----------------------------------------------------------------------------
# STEP 7: Use Data Sources to Query Existing Resources
# -----------------------------------------------------------------------------
# Data sources let you read information without creating anything.

data "dreamfactory_services" "all" {
  depends_on = [dreamfactory_service.customer_db]
}

# -----------------------------------------------------------------------------
# STEP 8: Define Outputs
# -----------------------------------------------------------------------------
# Outputs are displayed after 'terraform apply' and can be queried later
# with 'terraform output'. Great for getting API keys!

output "mobile_app_api_key" {
  value       = dreamfactory_app.mobile_app.api_key
  sensitive   = true
  description = "API key for the mobile app - use X-DreamFactory-API-Key header"
}

output "backend_service_api_key" {
  value       = dreamfactory_app.backend_service.api_key
  sensitive   = true
  description = "API key for the backend service"
}

output "public_api_key" {
  value       = dreamfactory_app.public_api.api_key
  sensitive   = true
  description = "API key for public API access"
}

output "customer_db_endpoint" {
  value       = "${var.dreamfactory_host}/api/v2/${dreamfactory_service.customer_db.name}"
  description = "Base URL for the customer database API"
}

output "all_services" {
  value       = [for s in data.dreamfactory_services.all.services : s.name]
  description = "List of all service names"
}

# =============================================================================
# HOW TO USE THIS:
# =============================================================================
#
# 1. First time setup:
#    cd examples/getting-started
#    terraform init
#
# 2. See what will be created:
#    terraform plan -var="dreamfactory_password=TerraformTest123!"
#
# 3. Create the resources:
#    terraform apply -var="dreamfactory_password=TerraformTest123!"
#
# 4. Get the API keys:
#    terraform output mobile_app_api_key
#    terraform output -json  # Get all outputs as JSON
#
# 5. Test the API:
#    API_KEY=$(terraform output -raw mobile_app_api_key)
#    curl -H "X-DreamFactory-API-Key: $API_KEY" http://localhost/api/v2/customers/_table
#
# 6. Make changes:
#    Edit this file, then run: terraform apply -var="dreamfactory_password=..."
#
# 7. Destroy everything:
#    terraform destroy -var="dreamfactory_password=TerraformTest123!"
#
# 8. See current state:
#    terraform show
#
# =============================================================================
