package datadog

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MLaureB/go-datadog-api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const datadogDashboardConfig = `
resource "datadog_dashboard" "ordered_dashboard" {
  title         = "Acceptance Test Ordered Dashboard"
  description   = "Created using the Datadog provider in Terraform"
  layout_type   = "ordered"
  is_read_only  = true

  widget {
  	note_definition {
      content = "note text"
      background_color = "pink"
      font_size = "14"
      text_align = "center"
    }
	}

	widget {
		group_definition {
			layout_type = "ordered"
			title = "Group Widget"

			widget {
				note_definition {
					content = "cluster note widget"
      		background_color = "yellow"
				}
			}

			widget {
				alert_graph_definition {
					alert_id = "123"
					viz_type = "toplist"
					title = "Alert Graph"
					title_size = "16"
					title_align = "right"
					time = {
						live_span = "1h"
					}
				}
			}
		}
	}

  template_variable {
    name   = "var_1"
    prefix = "host"
    default = "aws"
	}

	template_variable {
    name   = "var_2"
    prefix = "service_name"
    default = "autoscaling"
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
				Config: datadogDashboardConfig,
				Check: resource.ComposeTestCheckFunc(
					checkDashboardExists,
					// Dashboard metadata
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "title", "Acceptance Test Ordered Dashboard"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "description", "Created using the Datadog provider in Terraform"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "layout_type", "ordered"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "is_read_only", "true"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.#", "2"),
					// Note widget
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.0.note_definition.0.content", "note text"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.0.note_definition.0.background_color", "pink"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.0.note_definition.0.font_size", "14"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.0.note_definition.0.text_align", "center"),
					// Group widget
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.1.group_definition.0.layout_type", "ordered"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.1.group_definition.0.title", "Group Widget"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.1.group_definition.0.widget.#", "2"),
					// Inner Note widget
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.1.group_definition.0.widget.0.note_definition.0.content", "cluster note widget"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.1.group_definition.0.widget.0.note_definition.0.background_color", "yellow"),
					// Inner Alert Graph widget
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.1.group_definition.0.widget.1.alert_graph_definition.0.alert_id", "123"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.1.group_definition.0.widget.1.alert_graph_definition.0.viz_type", "toplist"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.1.group_definition.0.widget.1.alert_graph_definition.0.title", "Alert Graph"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.1.group_definition.0.widget.1.alert_graph_definition.0.title_size", "16"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.1.group_definition.0.widget.1.alert_graph_definition.0.title_align", "right"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.1.group_definition.0.widget.1.alert_graph_definition.0.time.live_span", "1h"),
					// Template Variables
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "template_variable.#", "2"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "template_variable.0.name", "var_1"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "template_variable.0.prefix", "host"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "template_variable.0.default", "aws"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "template_variable.1.name", "var_2"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "template_variable.1.prefix", "service_name"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "template_variable.1.default", "autoscaling"),
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
