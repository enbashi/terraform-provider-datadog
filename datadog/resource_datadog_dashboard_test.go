package datadog

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	datadog "github.com/zorkian/go-datadog-api"
)

const datadogDashboardConfig = `
resource "datadog_dashboard" "ordered_dashboard" {
  title         = "Acceptance Test Ordered Dashboard"
  description   = "Created using the Datadog provider in Terraform"
  layout_type   = "ordered"
  is_read_only  = true

	widget {
		alert_graph_definition {
			alert_id = "895605"
			viz_type = "timeseries"
			title = "Widget Title"
			title_size = 16
			title_align = "left"
			time = {
				live_span = "1h"
			}
		}
	}
	widget {
		alert_value_definition {
			alert_id = "895605"
			precision = 3
			unit = "b"
			text_align = "center"
			title = "Widget Title"
			title_size = 16
			title_align = "left"
		}
	}
	widget {
		change_definition {
			request {
				q = "avg:system.load.1{env:staging} by {account}"
				change_type = "absolute"
				compare_to = "week_before"
				increase_good = true
				order_by = "name"
				order_dir = "desc"
				show_present = true
			}
			title = "Widget Title"
			title_size = 16
			title_align = "left"
			time = {
				live_span = "1h"
			}
		}
	}
	widget {
		distribution_definition {
			request {
				q = "avg:system.load.1{env:staging} by {account}"
			}
			title = "Widget Title"
			title_size = 16
			title_align = "left"
			time = {
				live_span = "1h"
			}
		}
	}
	widget {
		check_status_definition {
			check = "aws.ecs.agent_connected"
			grouping = "cluster"
			group_by = ["account", "cluster"]
			tags = ["account:demo", "cluster:awseb-ruthebdog-env-8-dn3m6u3gvk"]
			title = "Widget Title"
			title_size = 16
			title_align = "left"
			time = {
				live_span = "1h"
			}
		}
	}
	widget {
		heatmap_definition {
			request {
				q = "avg:system.load.1{env:staging} by {account}"
			}
			yaxis = {
				min = 1
				max = 2
				include_zero = true
				scale = "sqrt"
			}
			title = "Widget Title"
			title_size = 16
			title_align = "left"
			time = {
				live_span = "1h"
			}
		}
	}
	widget {
		hostmap_definition {
			request {
				fill {
					q = "avg:system.load.1{*} by {host}"
				}
				size {
					q = "avg:memcache.uptime{*} by {host}"
				}
			}
			node_type= "container"
			group = ["host", "region"]
			no_group_hosts = true
			no_metric_hosts = true
			scope = ["region:us-east-1", "aws_account:727006795293"]
			title = "Widget Title"
			title_size = 16
			title_align = "left"
		}
	}
	widget {
		note_definition {
			content = "note text"
			background_color = "pink"
			font_size = "14"
			text_align = "center"
			show_tick = true
			tick_edge = "left"
			tick_pos = "50%"
		}
	}
	widget {
		query_value_definition {
		  request {
			q = "avg:system.load.1{env:staging} by {account}"
			aggregator = "sum"
			conditional_formats {
				comparator = "<"
				value = "2"
				palette = "white_on_green"
			}
			conditional_formats {
				comparator = ">"
				value = "2.2"
				palette = "white_on_red"
			}
		  }
		  autoscale = true
		  custom_unit = "xx"
		  precision = "4"
		  text_align = "right"
		  title = "Widget Title"
		  title_size = 16
		  title_align = "left"
		  time = {
			live_span = "1h"
		  }
		}
	}
	widget {
		scatterplot_definition {
			request {
				x {
					q = "avg:system.cpu.user{*} by {service, account}"
					aggregator = "max"
				}
				y {
					q = "avg:system.mem.used{*} by {service, account}"
					aggregator = "min"
				}
			}
			color_by_groups = ["account", "apm-role-group"]
			xaxis = {
				include_zero = true
				label = "x"
				min = "1"
				max = "2000"
				scale = "pow"
			}
			yaxis = {
				include_zero = false
				label = "y"
				min = "5"
				max = "2222"
				scale = "log"
			}
			title = "Widget Title"
			title_size = 16
			title_align = "left"
			time = {
				live_span = "1h"
			}
		}
	}
	widget {
		timeseries_definition {
			request {
				q= "avg:system.cpu.user{app:general} by {env}"
				display_type = "line"
			}
			request {
				log_query {
					index = "mcnulty"
					compute = {
						aggregation = "count"
						facet = "@duration"
						interval = 5000
					}
					search = {
						query = "status:info"
					}
					group_by {
						facet = "host"
						limit = 10
						sort = {
							aggregation = "avg"
							order = "desc"
							facet = "@duration"
						}
					}
				}
				display_type = "area"
			}
			request {
				apm_query {
					index = "apm-search"
					compute = {
						aggregation = "count"
						facet = "@duration"
						interval = 5000
					}
					search = {
						query = "type:web"
					}
					group_by {
						facet = "resource_name"
						limit = 50
						sort = {
							aggregation = "avg"
							order = "desc"
							facet = "@string_query.interval"
						}
					}
				}
				display_type = "bars"
			}
			request {
				process_query {
					metric = "process.stat.cpu.total_pct"
					search_by = "error"
					filter_by = ["active"]
					limit = 50
				}
				display_type = "area"
			}
			marker {
				display_type = "error dashed"
				label = " z=6 "
				value = "y = 4"
			}
			marker {
				display_type = "ok solid"
				value = "10 < y < 999"
				label = " x=8 "
			}
			title = "Widget Title"
			title_size = 16
			title_align = "left"
			time = {
				live_span = "1h"
			}
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
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.#", "14"),
					// Note widget
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.0.alert_graph_definition.0.content", "note text"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.0.alert_graph_definition.0.background_color", "pink"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.0.alert_graph_definition.0.font_size", "14"),
					resource.TestCheckResourceAttr("datadog_dashboard.ordered_dashboard", "widget.0.alert_graph_definition.0.text_align", "center"),
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

func TestAccDatadogDashboard_import(t *testing.T) {
	resourceName := "datadog_dashboard.ordered_dashboard"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkDashboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: datadogDashboardConfig,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
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
