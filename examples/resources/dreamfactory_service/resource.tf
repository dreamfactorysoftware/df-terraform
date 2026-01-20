# Example: Create a PostgreSQL database service
resource "dreamfactory_service" "postgres_db" {
  name        = "customer_db"
  label       = "Customer Database"
  description = "PostgreSQL database containing customer data"
  type        = "pgsql"
  is_active   = true

  host     = "postgres.example.com"
  port     = 5432
  database = "customers"
  username = "app_user"
  password = var.postgres_password
}

# Example: Create a MySQL database service
resource "dreamfactory_service" "mysql_db" {
  name        = "orders_db"
  label       = "Orders Database"
  description = "MySQL database for order management"
  type        = "mysql"
  is_active   = true

  host     = "mysql.example.com"
  port     = 3306
  database = "orders"
  username = "orders_user"
  password = var.mysql_password
}

# Example: Create a MongoDB service
resource "dreamfactory_service" "mongodb" {
  name        = "analytics_db"
  label       = "Analytics Database"
  description = "MongoDB for analytics data"
  type        = "mongodb"
  is_active   = true

  host     = "mongodb.example.com"
  port     = 27017
  database = "analytics"
  username = "analytics_user"
  password = var.mongodb_password
}

variable "postgres_password" {
  type      = string
  sensitive = true
}

variable "mysql_password" {
  type      = string
  sensitive = true
}

variable "mongodb_password" {
  type      = string
  sensitive = true
}
