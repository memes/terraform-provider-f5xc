package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBlindfoldResource(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "f5xc_blindfold" "test" {
	plaintext = "VGhpcyBpcyBhIHRlc3Q="
	policy_document = {
		name = "ves-io-allow-volterra"
		namespace = "shared"
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("f5xc_blindfold.test", "id"),
					resource.TestCheckResourceAttr("f5xc_blindfold.test", "plaintext", "VGhpcyBpcyBhIHRlc3Q="),
					resource.TestCheckResourceAttrSet("f5xc_blindfold.test", "sealed"),
				),
			},
		},
	})
}
