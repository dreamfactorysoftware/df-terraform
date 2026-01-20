terraform {
  required_providers {
    dreamfactory = {
      source  = "dreamfactorysoftware/dreamfactory"
      version = "~> 0.1.0"
    }
  }
}

provider "dreamfactory" {
  host     = "http://localhost"
  email    = "admin@example.com"
  password = var.dreamfactory_password
}

variable "dreamfactory_password" {
  type        = string
  sensitive   = true
  description = "Password for the DreamFactory admin user"
}
