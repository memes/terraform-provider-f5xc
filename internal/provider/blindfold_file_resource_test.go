package provider_test

import (
	"errors"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBlindfoldFileResource(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "plaintext")
	if err != nil {
		t.Errorf("failed to create temporary plaintext file: %v", err)
	}
	defer func() {
		if err := tmpFile.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			t.Errorf("error closing tmpFile: %v", err)
		}
	}()
	if _, err = tmpFile.Write([]byte("This is a plaintext document to be blindfolded")); err != nil {
		t.Errorf("failed to write data to plaintext file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Errorf("failed to close plaintext file: %v", err)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "f5xc_blindfold_file" "test" {
	path = "` + tmpFile.Name() + `"
	policy_document = {
		name = "ves-io-allow-volterra"
		namespace = "shared"
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("f5xc_blindfold_file.test", "id"),
					resource.TestCheckResourceAttr("f5xc_blindfold_file.test", "path", tmpFile.Name()),
					resource.TestCheckResourceAttrSet("f5xc_blindfold_file.test", "sealed"),
				),
			},
		},
	})
}
