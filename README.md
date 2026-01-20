# DreamFactory Terraform Provider

Official Terraform provider for [DreamFactory](https://www.dreamfactory.com/) - manage your DreamFactory instance as Infrastructure as Code.

## Features

- **Services** - Create and manage database connections that become REST APIs
- **Roles** - Define access control policies for API consumers
- **Apps** - Generate and manage API keys for authentication

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (for building from source)
- A running DreamFactory instance (v4.x or v5.x)

## Installation

### From Source

```bash
git clone https://github.com/dreamfactorysoftware/df-terraform.git
cd df-terraform
go build -o terraform-provider-dreamfactory

# Install to local plugins directory
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/dreamfactorysoftware/dreamfactory/0.1.0/linux_amd64
cp terraform-provider-dreamfactory ~/.terraform.d/plugins/registry.terraform.io/dreamfactorysoftware/dreamfactory/0.1.0/linux_amd64/
```

For macOS, replace `linux_amd64` with `darwin_amd64` or `darwin_arm64`.

## Quick Start

```hcl
terraform {
  required_providers {
    dreamfactory = {
      source  = "registry.terraform.io/dreamfactorysoftware/dreamfactory"
      version = "0.1.0"
    }
  }
}

provider "dreamfactory" {
  host     = "https://api.example.com"
  email    = "admin@example.com"
  password = var.admin_password
}

# Create a database service
resource "dreamfactory_service" "mydb" {
  name     = "mydb"
  label    = "My Database"
  type     = "mysql"
  host     = "mysql.example.com"
  port     = 3306
  database = "myapp"
  username = "dbuser"
  password = var.db_password
}

# Create a role
resource "dreamfactory_role" "api_users" {
  name        = "api_users"
  description = "External API consumers"
}

# Create an app (API key)
resource "dreamfactory_app" "my_app" {
  name    = "my_app"
  role_id = dreamfactory_role.api_users.id
}

# Get the generated API key
output "api_key" {
  value     = dreamfactory_app.my_app.api_key
  sensitive = true
}
```

Run:

```bash
terraform init
terraform plan -var="admin_password=..." -var="db_password=..."
terraform apply -var="admin_password=..." -var="db_password=..."

# Get the API key
terraform output -raw api_key
```

## Provider Configuration

```hcl
provider "dreamfactory" {
  host                 = "https://api.example.com"  # Required
  email                = "admin@example.com"        # Required
  password             = var.password               # Required
  insecure_skip_verify = false                      # Optional: skip TLS verification
}
```

### Environment Variables

```bash
export DREAMFACTORY_HOST="https://api.example.com"
export DREAMFACTORY_EMAIL="admin@example.com"
export DREAMFACTORY_PASSWORD="your-password"
```

## Resources

### dreamfactory_service

Creates a database service that exposes tables as REST API endpoints.

```hcl
resource "dreamfactory_service" "example" {
  name        = "customers"           # Required: URL-safe identifier
  type        = "mysql"               # Required: Database type
  label       = "Customer DB"         # Optional: Display name
  description = "Customer database"   # Optional
  is_active   = true                  # Optional: Default true

  # Database connection
  host     = "db.example.com"
  port     = 3306
  database = "customers"
  username = "app_user"
  password = var.db_password
  schema   = "public"                 # Optional: For PostgreSQL/SQL Server
}
```

**Supported database types:**

| Type | Database |
|------|----------|
| `mysql` | MySQL, MariaDB |
| `pgsql` | PostgreSQL |
| `sqlsrv` | Microsoft SQL Server |
| `oracle` | Oracle |
| `sqlite` | SQLite |
| `mongodb` | MongoDB |
| `couchdb` | CouchDB |
| `aws_dynamodb` | AWS DynamoDB |
| `snowflake` | Snowflake |

### dreamfactory_role

Creates a role for access control.

```hcl
resource "dreamfactory_role" "api_users" {
  name        = "api_users"           # Required
  description = "API user role"       # Optional
  is_active   = true                  # Optional: Default true
}
```

**Note:** Role permissions (service/table access) must be configured in the DreamFactory UI.

### dreamfactory_app

Creates an app which generates an API key.

```hcl
resource "dreamfactory_app" "mobile" {
  name        = "mobile_app"                       # Required
  description = "Mobile app API key"               # Optional
  is_active   = true                               # Optional: Default true
  type        = 0                                  # Optional: 0=No storage
  role_id     = dreamfactory_role.api_users.id    # Optional: Assign role
}

output "api_key" {
  value     = dreamfactory_app.mobile.api_key
  sensitive = true
}
```

## Data Sources

### dreamfactory_services

Read-only query to list all services.

```hcl
data "dreamfactory_services" "all" {}

output "service_names" {
  value = [for s in data.dreamfactory_services.all.services : s.name]
}
```

## Importing Existing Resources

```bash
terraform import dreamfactory_service.mydb 8
terraform import dreamfactory_role.myrole 5
terraform import dreamfactory_app.myapp 12
```

Find resource IDs in the DreamFactory admin UI or via API.

## Security

- **TLS 1.2 minimum** enforced by default
- Passwords and API keys marked as sensitive (not shown in logs)
- Support for `insecure_skip_verify` for development with self-signed certs
- Environment variable support for credentials

**Best Practices:**

- Never hardcode passwords in `.tf` files
- Use variables with `sensitive = true`
- Secure your state file (contains API keys)
- Use HTTPS in production

## Development

### Building

```bash
go mod tidy
go build -o terraform-provider-dreamfactory
```

### Testing

```bash
# Unit tests
go test ./...

# With verbose output
go test -v ./...

# Acceptance tests (requires running DreamFactory)
export DREAMFACTORY_HOST="http://localhost"
export DREAMFACTORY_EMAIL="admin@example.com"
export DREAMFACTORY_PASSWORD="your-password"
TF_ACC=1 go test -v ./internal/provider/... -run TestAcc
```

### Project Structure

```
df-terraform/
├── main.go                       # Entry point
├── go.mod                        # Go module
├── internal/
│   ├── client/
│   │   ├── client.go             # DreamFactory API client
│   │   └── client_test.go        # Client tests
│   └── provider/
│       ├── provider.go           # Provider configuration
│       ├── validators.go         # Input validators
│       ├── service_resource.go   # Service resource
│       ├── role_resource.go      # Role resource
│       ├── app_resource.go       # App resource
│       └── services_data_source.go
└── examples/
    └── getting-started/
        └── main.tf               # Example configuration
```

## Compatibility

Tested with:
- DreamFactory 4.x
- DreamFactory 5.x
- Terraform 1.0+

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.

## Support

- [DreamFactory Documentation](https://wiki.dreamfactory.com/)
- [GitHub Issues](https://github.com/dreamfactorysoftware/df-terraform/issues)
- [DreamFactory Support](https://www.dreamfactory.com/support/)
