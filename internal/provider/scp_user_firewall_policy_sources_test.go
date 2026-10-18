package provider_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccScpUserFirewallPolicy_addressSets covers both address fields across
// create, config reordering, update and refresh with alternating API order.
func TestAccScpUserFirewallPolicy_addressSets(t *testing.T) {
	for _, field := range []string{"sources", "destinations"} {
		t.Run(field, func(t *testing.T) {
			server := newFirewallPolicyMockServer(t)
			defer server.Close()
			config := firewallPolicyRuleConfig(server.URL, "multi", fmt.Sprintf(`
      %s = ["192.0.2.10/32", "192.0.2.20/32", "192.0.2.30/32"]`, field))
			reordered := firewallPolicyRuleConfig(server.URL, "multi", fmt.Sprintf(`
      %s = ["192.0.2.30/32", "192.0.2.10/32", "192.0.2.20/32"]`, field))
			updated := firewallPolicyRuleConfig(server.URL, "updated", fmt.Sprintf(`
      %s = ["192.0.2.40/32", "192.0.2.20/32"]`, field))
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: config, Check: firewallAddressSetCheck(field, "192.0.2.10/32", "192.0.2.20/32", "192.0.2.30/32")},
					{Config: reordered, PlanOnly: true},
					{Config: updated, Check: firewallAddressSetCheck(field, "192.0.2.40/32", "192.0.2.20/32")},
					{Config: updated, PlanOnly: true},
				},
			})
		})
	}
}

func firewallAddressSetCheck(field string, addresses ...string) resource.TestCheckFunc {
	checks := make([]resource.TestCheckFunc, 0, 1+len(addresses))
	checks = append(checks, resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "rules.0."+field+".#", strconv.Itoa(len(addresses))))
	for _, address := range addresses {
		checks = append(checks, resource.TestCheckTypeSetElemAttr("netcup_scp_user_firewall_policy.test", "rules.0."+field+".*", address))
	}
	return resource.ComposeAggregateTestCheckFunc(checks...)
}

// TestAccScpUserFirewallPolicy_upgradeFromListState creates state with the
// released v1.2.1 provider, where sources/destinations were lists, then plans
// with this build, where they are sets. The plan must be empty and must not
// error while decoding the existing state.
func TestAccScpUserFirewallPolicy_upgradeFromListState(t *testing.T) {
	server := newFirewallPolicyMockServer(t)
	defer server.Close()

	config := firewallPolicyRuleConfig(server.URL, "upgrade", `
      sources      = ["192.0.2.10/32"]
      destinations = ["192.0.2.1/32"]`)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"netcup": {Source: "rixlhq/netcup", VersionConstraint: "1.2.1"},
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "rules.0.sources.#", "1"),
					resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "rules.0.destinations.#", "1"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				PlanOnly:                 true,
			},
		},
	})
}

// TestAccScpUserFirewallPolicy_emptyAndNullAddresses checks both fields with
// explicit empty, omitted and null values, each followed by an empty plan.
func TestAccScpUserFirewallPolicy_emptyAndNullAddresses(t *testing.T) {
	for _, field := range []string{"sources", "destinations"} {
		t.Run(field, func(t *testing.T) {
			server := newFirewallPolicyMockServer(t)
			defer server.Close()
			empty := firewallPolicyRuleConfig(server.URL, "empty", "\n      "+field+" = []")
			omitted := firewallPolicyRuleConfig(server.URL, "omitted", "")
			null := firewallPolicyRuleConfig(server.URL, "null", "\n      "+field+" = null")
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: empty, Check: firewallAddressSetCheck(field)},
					{Config: empty, PlanOnly: true},
					{Config: omitted, Check: resource.TestCheckNoResourceAttr("netcup_scp_user_firewall_policy.test", "rules.0."+field+".#")},
					{Config: omitted, PlanOnly: true},
					{Config: null, Check: resource.TestCheckNoResourceAttr("netcup_scp_user_firewall_policy.test", "rules.0."+field+".#")},
					{Config: null, PlanOnly: true},
				},
			})
		})
	}
}

// firewallPolicyRuleConfig renders a single-rule policy against the mock at
// serverURL. extraAttrs is appended verbatim inside the rule block.
func firewallPolicyRuleConfig(serverURL, desc, extraAttrs string) string {
	return fmt.Sprintf(`
provider "netcup" {
  scp_access_token = "mock-token"
  scp_base_url     = "%s/scp-core"
}

resource "netcup_scp_user_firewall_policy" "test" {
  user_id     = 12345
  name        = "test-policy"
  description = %q
  rules = [
    {
      action            = "ACCEPT"
      direction         = "INGRESS"
      protocol          = "TCP"
      destination_ports = "443"%s
    }
  ]
}
`, serverURL, desc, extraAttrs)
}
