package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDNSCredentials_scpOnly(t *testing.T) {
	configs := map[string]string{
		"record": `resource "netcup_dns_record" "test" {
  zone = "example.com"
  hostname = "www"
  type = "A"
  destination = "192.0.2.1"
}`,
		"zone": `resource "netcup_dns_zone" "test" {
  domain_name = "example.com"
  ttl = 3600
}`,
		"records data source": `data "netcup_dns_records" "test" { domain_name = "example.com" }`,
		"zone data source":    `data "netcup_dns_zone" "test" { domain_name = "example.com" }`,
	}
	for name, config := range configs {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:      `provider "netcup" { scp_access_token = "mock-token" }` + "\n" + config,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile("Missing CCP Credentials"),
				}},
			})
		})
	}
}
