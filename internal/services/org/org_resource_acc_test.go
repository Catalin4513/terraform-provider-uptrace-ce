package org_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	upClient "github.com/catalin4513/terraform-provider-uptrace-ce/internal/client"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/generated"
	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/testutil"
)

func testAccOrgConfig(name string) string {
	return fmt.Sprintf(`
resource "uptrace_org" "test" {
  name = %q
}
`, name)
}

func testAccOrgConfigWithBudget(name string, budget float64) string {
	return fmt.Sprintf(`
resource "uptrace_org" "test" {
  name   = %q
  budget = %v
}
`, name, budget)
}

func testAccCheckOrgDestroy(s *terraform.State) error {
	c := upClient.New(
		os.Getenv("UPTRACE_ENDPOINT"),
		os.Getenv("UPTRACE_TOKEN"),
		0,
	)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptrace_org" {
			continue
		}
		orgID, err := strconv.ParseUint(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid org ID %q: %w", rs.Primary.ID, err)
		}
		_, err = c.API.GetOrg(context.Background(), &generated.GetOrgRequestOptions{
			PathParams: &generated.GetOrgPath{OrgID: orgID},
		})
		if err == nil {
			return fmt.Errorf("org %s still exists after destroy", rs.Primary.ID)
		}
	}
	return nil
}

func TestAccOrg_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckOrgDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOrgConfig("acc-test-org"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_org.test", "id"),
					resource.TestCheckResourceAttr("uptrace_org.test", "name", "acc-test-org"),
					resource.TestCheckResourceAttrSet("uptrace_org.test", "budget"),
				),
			},
			{
				Config: testAccOrgConfig("acc-test-org-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptrace_org.test", "name", "acc-test-org-renamed"),
				),
			},
			{
				ResourceName:      "uptrace_org.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOrg_withBudget(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.PreCheck(t) },
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckOrgDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOrgConfigWithBudget("acc-budget-org", 250),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptrace_org.test", "id"),
					resource.TestCheckResourceAttr("uptrace_org.test", "name", "acc-budget-org"),
					resource.TestCheckResourceAttr("uptrace_org.test", "budget", "250"),
				),
			},
			{
				Config: testAccOrgConfigWithBudget("acc-budget-org", 500),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptrace_org.test", "budget", "500"),
				),
			},
			{
				ResourceName:      "uptrace_org.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
