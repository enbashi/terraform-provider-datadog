package datadog

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zorkian/go-datadog-api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const testAccCheckDatadogDashboard = `
resource "datadog_dashboard" "acceptance_test" {
  title         = "Acceptance Test Dashboard"
  description   = "Created using the Datadog provider in Terraform"
  layout_type   = "free"
  is_read_only  = true

  widget {
    definition {
      type = "note"
      background_color = "pink"
      font_size = "14"
      content = "note text"
      text_align = "center"
      show_tick = "false"
      tick_edge = "bottom"
      tick_pos = "50%"
    }
    layout {
        x = 36
        y = 25
        width = 140
        height = 500
    }
  }

  template_variable {
    name   = "var"
    prefix = "host"
    default = "aws"
  }
}
`

func TestAccDatadogDashboard_update(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkDashboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDatadogDashboard,
				Check: resource.ComposeTestCheckFunc(
					checkDashboardExists,
					resource.TestCheckResourceAttr("datadog_dashboard.acceptance_test", "title", "Acceptance Test Dashboard"),
					resource.TestCheckResourceAttr("datadog_dashboard.acceptance_test", "layout_type", "free"),
					resource.TestCheckResourceAttr("datadog_dashboard.acceptance_test", "description", "Created using the Datadog provider in Terraform"),
					resource.TestCheckResourceAttr("datadog_dashboard.acceptance_test", "is_read_only", "true"),
					resource.TestCheckResourceAttr("datadog_dashboard.acceptance_test", "widget.0.definition.type", "note"),
					resource.TestCheckResourceAttr("datadog_dashboard.acceptance_test", "widget.0.definition.background_color", "pink"),
					resource.TestCheckResourceAttr("datadog_dashboard.acceptance_test", "widget.0.definition.font_size", "14"),
					resource.TestCheckResourceAttr("datadog_dashboard.acceptance_test", "widget.0.definition.content", "note text"),
					resource.TestCheckResourceAttr("datadog_dashboard.acceptance_test", "widget.0.definition.text_align", "center"),
					resource.TestCheckResourceAttr("datadog_dashboard.acceptance_test", "widget.0.definition.show_tick", "false"),
					resource.TestCheckResourceAttr("datadog_dashboard.acceptance_test", "widget.0.definition.tick_edge", "bottom"),
					resource.TestCheckResourceAttr("datadog_dashboard.acceptance_test", "widget.0.definition.tick_pos", "50%"),
				),
			},
		},
	})
}

func checkDashboardExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*datadog.Client)
	for _, r := range s.RootModule().Resources {
		if _, err := client.GetBoard(r.Primary.ID); err != nil {
			return fmt.Errorf("Received an error retrieving dashboard1 %s", err)
		}
	}
	return nil
}

func checkDashboardDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*datadog.Client)
	for _, r := range s.RootModule().Resources {
		if _, err := client.GetBoard(r.Primary.ID); err != nil {
			if strings.Contains(err.Error(), "404 Not Found") {
				continue
			}
			return fmt.Errorf("Received an error retrieving dashboard2 %s", err)
		}
		return fmt.Errorf("Timeboard still exists")
	}
	return nil
}
