package datadog

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/MLaureB/go-datadog-api"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/kr/pretty"
)

func resourceDatadogDashboard() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogDashboardCreate,
		Update: resourceDatadogDashboardUpdate,
		Read:   resourceDatadogDashboardRead,
		Delete: resourceDatadogDashboardDelete,
		Exists: resourceDatadogDashboardExists,
		Importer: &schema.ResourceImporter{
			State: resourceDatadogDashboardImport,
		},
		Schema: map[string]*schema.Schema{
			"title": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The title of the dashboard.",
			},
			"widget_json": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"definition": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			// "widget": {
			// 	Type:        schema.TypeList,
			// 	Required:    true,
			// 	Description: "The list of widgets to display on the dashboard.",
			// 	Elem: &schema.Resource{
			// 		Schema: getWidgetSchema(),
			// 	},
			// },
			"layout_type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The layout type of the dashboard, either 'free' or 'ordered'.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the dashboard.",
			},
			"is_read_only": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether this dashboard is read-only.",
			},
			"template_variable": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The list of template variables for this dashboard.",
				Elem: &schema.Resource{
					Schema: getTemplateVariableSchema(),
				},
			},
			"notify_list": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The list of handles of users to notify when changes are made to this dashboard.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func buildDatadogDashboard(d *schema.ResourceData) (*datadog.Board, error) {
	// Build Dashboard metadata
	dashboard := datadog.Board{
		Id:          datadog.String(d.Id()),
		Title:       datadog.String(d.Get("title").(string)),
		LayoutType:  datadog.String(d.Get("layout_type").(string)),
		Description: datadog.String(d.Get("description").(string)),
		IsReadOnly:  datadog.Bool(d.Get("is_read_only").(bool)),
	}

	// Build Widgets with JSON
	terraformWidgets := d.Get("widget_json").([]interface{})
	datadogWidgets := make([]datadog.BoardWidget, len(terraformWidgets))
	for i, _terraformWidget := range terraformWidgets {
		terraformWidget := _terraformWidget.(map[string]interface{})
		datadogWidget := datadog.BoardWidget{}
		// Build Datadog definition
		terraformDefinition := terraformWidget["definition"].(string)
		noteDefinition := datadog.NoteDefinition{}
		if err := json.Unmarshal([]byte(terraformDefinition), &noteDefinition); err != nil {
			return nil, err
		}
		datadogWidget.Definition = noteDefinition
		datadogWidgets[i] = datadogWidget
	}
	dashboard.Widgets = datadogWidgets

	// // Build Widgets
	// widgets := d.Get("widget").([]interface{})
	// datadogWidgets, err := buildDatadogWidgets(&widgets)
	// if err != nil {
	// 	return nil, err
	// }
	// dashboard.Widgets = *datadogWidgets

	// Build NotifyList
	notifyList := d.Get("notify_list").([]interface{})
	dashboard.NotifyList = buildDatadogNotifyList(&notifyList)

	// Build TemplateVariables
	templateVariables := d.Get("template_variable").([]interface{})
	dashboard.TemplateVariables = *buildDatadogTemplateVariables(&templateVariables)

	return &dashboard, nil
}

func resourceDatadogDashboardCreate(d *schema.ResourceData, meta interface{}) error {
	dashboard, err := buildDatadogDashboard(d)
	if err != nil {
		return fmt.Errorf("Failed to parse resource configuration: %s", err.Error())
	}
	dashboard, err = meta.(*datadog.Client).CreateBoard(dashboard)
	if err != nil {
		return fmt.Errorf("Failed to create dashboard using Datadog API: %s", err.Error())
	}
	d.SetId(*dashboard.Id)
	return nil
}

func resourceDatadogDashboardUpdate(d *schema.ResourceData, meta interface{}) error {
	dashboard, err := buildDatadogDashboard(d)
	if err != nil {
		return fmt.Errorf("Failed to parse resource configuration: %s", err.Error())
	}
	if err = meta.(*datadog.Client).UpdateBoard(dashboard); err != nil {
		return fmt.Errorf("Failed to update dashboard using Datadog API: %s", err.Error())
	}
	return resourceDatadogDashboardRead(d, meta)
}

func resourceDatadogDashboardRead(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()
	dashboard, err := meta.(*datadog.Client).GetBoard(id)
	if err != nil {
		return err
	}
	log.Printf("[DataDog] dashboard: %v", pretty.Sprint(dashboard))

	// Set title
	if err := d.Set("title", dashboard.Title); err != nil {
		return err
	}
	// Set JSON widgets
	// Build empty terraform widgets
	// for min(len(terraform
	// compare the widgets
	// 		if same: keep the terraforme one
	// 		if different: serialize the datadog one
	// if more widgets in datadogWidgets: serialize them all
	// set terraform widgets

	// Set widgets with JSON
	existingTerraformWidgets := d.Get("widget_json").([]interface{})
	datadogWidgets := &dashboard.Widgets
	terraformWidgets := make([]map[string]interface{}, len(*datadogWidgets))
	for i, datadogWidget := range *datadogWidgets {
		existingTerraformWidget := existingTerraformWidgets[i].(map[string]interface{})
		existingDefinition := existingTerraformWidget["definition"].(string)
		noteDefinition := datadog.NoteDefinition{}
		if err := json.Unmarshal([]byte(existingDefinition), &noteDefinition); err != nil {
			return err
		}
		terraformWidget := map[string]interface{}{}
		// return fmt.Errorf("Failed to compare terraform %s and datadog %s", pretty.Sprint(noteDefinition), pretty.Sprint(datadogWidget.Definition))
		// return fmt.Errorf("Failed to comparison:  ", reflect.DeepEqual(noteDefinition, datadogWidget.Definition))
		if reflect.DeepEqual(noteDefinition, datadogWidget.Definition) == false {
			// Store new datadog definition
			terraformDefinition, _ := json.Marshal(datadogWidget.Definition)
			terraformWidget["definition"] = string(terraformDefinition)
		} else {
			// Keep existing terraform definition
			terraformWidget["definition"] = existingDefinition
		}
		terraformWidgets[i] = terraformWidget
	}
	if err := d.Set("widget_json", terraformWidgets); err != nil {
		return err
	}

	// Set widgets
	// widgets, err := buildTerraformWidgets(&dashboard.Widgets)
	// if err != nil {
	// 	return err
	// }
	// if err := d.Set("widget", widgets); err != nil {
	// 	return err
	// }
	// Set layout type
	if err := d.Set("layout_type", dashboard.LayoutType); err != nil {
		return err
	}
	// Set description
	if err := d.Set("description", dashboard.Description); err != nil {
		return err
	}
	// Set is_read_only
	if err := d.Set("is_read_only", dashboard.IsReadOnly); err != nil {
		return err
	}
	// Set template variables
	templateVariables := buildTerraformTemplateVariables(&dashboard.TemplateVariables)
	if err := d.Set("template_variable", templateVariables); err != nil {
		return err
	}
	// Set notify list
	notifyList := buildTerraformNotifyList(&dashboard.NotifyList)
	if err := d.Set("notify_list", notifyList); err != nil {
		return err
	}

	return nil
}

func resourceDatadogDashboardDelete(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()
	if err := meta.(*datadog.Client).DeleteBoard(id); err != nil {
		return err
	}
	return nil
}

func resourceDatadogDashboardImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourceDatadogDashboardRead(d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func resourceDatadogDashboardExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	id := d.Id()
	if _, err := meta.(*datadog.Client).GetBoard(id); err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
