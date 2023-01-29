package provider_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDomainResource(t *testing.T) {
	domain := fmt.Sprintf("%s-test-terraform-provider.giautm.xyz", uuid.New())

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"cloudflare": {
				Source:            "cloudflare/cloudflare",
				VersionConstraint: "3.33.1",
			},
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainResourceConfig(domain),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googlesiteverification_domain.example", "domain", domain),
					resource.TestMatchResourceAttr("googlesiteverification_domain.example", "token", regexp.MustCompile(`^google-site-verification=[A-Za-z0-9_-]+$`)),
				),
			},
		},
	})
}

func testAccDomainResourceConfig(domain string) string {
	return fmt.Sprintf(`
	data "googlesiteverification_domain" "example" {
		id = %[1]q
	}
	resource "cloudflare_record" "verification" {
		zone_id = %[2]q
		name    = data.googlesiteverification_domain.example.record_name
		value   = data.googlesiteverification_domain.example.record_value
		type    = data.googlesiteverification_domain.example.record_type
	}
	resource "googlesiteverification_domain" "example" {
		domain = %[1]q
		token  = data.googlesiteverification_domain.example.record_value
		depends_on = [
			cloudflare_record.verification,
		]
		timeouts {
			create = "5m"
			delete = "15m"
		}
	}`, domain, os.Getenv("CLOUDFLARE_ZONE_ID"))
}
