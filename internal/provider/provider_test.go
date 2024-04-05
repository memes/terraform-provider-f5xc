package provider_test

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/memes/terraform-provider-f5xc/internal/provider"
)

const (
	providerConfig = `
provider "f5xc" {
  api_token = "invalid"
  api_url = "https://f5.invalid/api"
}
`
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
//
//nolint:gochecknoglobals // Shared so all provider_test functions can create a provider per test case.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"f5xc": providerserver.NewProtocol6WithError(provider.New("test")()),
}
