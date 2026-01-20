package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"dreamfactory": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// Check that required environment variables are set
	if v := os.Getenv("DREAMFACTORY_HOST"); v == "" {
		t.Fatal("DREAMFACTORY_HOST must be set for acceptance tests")
	}
	if v := os.Getenv("DREAMFACTORY_EMAIL"); v == "" {
		t.Fatal("DREAMFACTORY_EMAIL must be set for acceptance tests")
	}
	if v := os.Getenv("DREAMFACTORY_PASSWORD"); v == "" {
		t.Fatal("DREAMFACTORY_PASSWORD must be set for acceptance tests")
	}
}

// Helper function to get provider config from environment
func testAccProviderConfig() string {
	return `
provider "dreamfactory" {
  host     = "` + os.Getenv("DREAMFACTORY_HOST") + `"
  email    = "` + os.Getenv("DREAMFACTORY_EMAIL") + `"
  password = "` + os.Getenv("DREAMFACTORY_PASSWORD") + `"
}
`
}
