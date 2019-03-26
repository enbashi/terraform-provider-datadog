package datadog

import (
	// "encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/MLaureB/go-datadog-api"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/kr/pretty"
)

func resourceDatadogDashboard() *schema.Resource {

	widgetLayout := &schema.Schema{
		Type:        schema.TypeMap,
		Optional:    true,
		Description: "The layout for a widget on a 'free' dashboard.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"x": {
					Type:     schema.TypeFloat,
					Required: true,
				},
				"y": {
					Type:     schema.TypeFloat,
					Required: true,
				},
				"width": {
					Type:     schema.TypeFloat,
					Required: true,
				},
				"height": {
					Type:     schema.TypeFloat,
					Required: true,
				},
			},
		},
	}

	// shared definition props for all widgets
	widgetDefinitionSchema := map[string]*schema.Schema{
		"type": {
			Type:     schema.TypeString,
			Required: true,
		},
		"title": {
			Type:     schema.TypeString,
			Optional: true,
		},

		// Note widget
		"content": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"background_color": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"font_size": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"text_align": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"show_tick": {
			Type:     schema.TypeBool,
			Optional: true,
		},
		"tick_pos": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"tick_edge": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}

	// Clone widgetDefinitionSchema and reuse it for rootWidgetDefinitionSchema
	rootWidgetDefinitionSchema := make(map[string]*schema.Schema, len(widgetDefinitionSchema))
	for k, v := range widgetDefinitionSchema {
		rootWidgetDefinitionSchema[k] = v
	}

	// Additional props used in Group widget
	rootWidgetDefinitionSchema["layout_type"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The layout type of the group widget. Only 'ordered' is supported",
		ValidateFunc: validateGroupWidgetLayoutType,
	}

	groupWidget := &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "A list of widget definitions.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:        schema.TypeInt,
					Optional:    true,
					Description: "The id of the widget.",
				},
				"definition": {
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Description: "The definition of a widget.",
					Elem: &schema.Resource{
						Schema: widgetDefinitionSchema,
					},
				},
			},
		},
	}

	// Allow Group widget at the root level only
	rootWidgetDefinitionSchema["widget"] = groupWidget

	widget := &schema.Schema{
		Type:        schema.TypeList,
		Required:    true,
		Description: "A list of widget definitions.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:        schema.TypeInt,
					Optional:    true,
					Description: "The id of the widget.",
				},
				"layout": widgetLayout,
				"definition": {
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Description: "The definition of a widget.",
					Elem: &schema.Resource{
						Schema: rootWidgetDefinitionSchema,
					},
				},
			},
		},
	}

	templateVariable := &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "A list of template variables for using Dashboard templating.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The name of the variable.",
				},
				"prefix": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The tag prefix associated with the variable. Only tags with this prefix will appear in the variable dropdown.",
				},
				"default": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The default value for the template variable on dashboard load.",
				},
			},
		},
	}

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
				Description: "The name of the dashboard.",
			},
			"widget": widget,
			"layout_type": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The layout type of the dashboard.'free' or 'ordered'",
				ValidateFunc: validateDashboardLayoutType,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the dashboard's content.",
			},
			"is_read_only": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"template_variable": templateVariable,
			"author_handle": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The handle of the dashboard's author",
			},
			"notify_list": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The handle of the dashboard's author",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The URL of the dashboard",
			},
			"created_at": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The dashboard's creation time",
			},
			"modified_at": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The dashboard's last modification time",
			},
		},
	}
}

func buildDashboardWidgets(terraformWidgets *[]interface{}, layoutType string) (*[]datadog.BoardWidget, error) {
	datadogWidgets := make([]datadog.BoardWidget, len(*terraformWidgets))
	for i, widget := range *terraformWidgets {
		widgetMap := widget.(map[string]interface{})
		// The definition is defined in the schema as a TypeList with a MaxItems of 1
		// So we we need to read it as an array
		// This is in order to allow nested structure
		widgetDefinition := widgetMap["definition"].([]interface{})[0].(map[string]interface{})
		widgetType := datadog.String(widgetDefinition["type"].(string))

		datadogWidgets[i] = datadog.BoardWidget{}
		d := &datadogWidgets[i]

		if v, ok := widgetMap["id"]; ok {
			d.Id = datadog.Int(v.(int))
		}

		if layoutType == "free" {
			layoutMap := widgetMap["layout"].(map[string]interface{})
			layout := &datadog.WidgetLayout{}
			if v, err := strconv.ParseFloat(layoutMap["x"].(string), 64); err == nil {
				layout.X = &v
			}
			if v, err := strconv.ParseFloat(layoutMap["y"].(string), 64); err == nil {
				layout.Y = &v
			}
			if v, err := strconv.ParseFloat(layoutMap["height"].(string), 64); err == nil {
				layout.Height = &v
			}
			if v, err := strconv.ParseFloat(layoutMap["width"].(string), 64); err == nil {
				layout.Width = &v
			}
			d.Layout = layout
		}

		switch *widgetType {
		case "group":
			definition := &datadog.GroupDefinition{}

			// Required params
			definition.Type = widgetType
			if v, ok := widgetDefinition["layout_type"]; ok {
				definition.LayoutType = datadog.String(v.(string))
			}

			if v, ok := widgetDefinition["widget"].([]interface{}); ok {
				groupWidgets, err := buildDashboardWidgets(&v, "ordered")
				if err != nil {
					return nil, fmt.Errorf("Failed to parse group widget: %s", err.Error())
				}
				definition.Widgets = *groupWidgets
			}
			// Optional params
			if v, ok := widgetDefinition["title"]; ok {
				WidgetTitle := datadog.WidgetTitle(v.(string))
				definition.Title = &WidgetTitle
			}

			d.Definition = definition

		case "note":
			definition := &datadog.NoteDefinition{}

			// Required params
			definition.Type = widgetType
			if v, ok := widgetDefinition["content"].(string); ok && len(v) != 0 {
				definition.Content = datadog.String(v)
			}
			// Optional params
			if v, ok := widgetDefinition["background_color"].(string); ok && len(v) != 0 {
				definition.BackgroundColor = datadog.String(v)
			}
			if v, ok := widgetDefinition["font_size"].(string); ok && len(v) != 0 {
				definition.FontSize = datadog.String(v)
			}
			if v, ok := widgetDefinition["text_align"].(string); ok && len(v) != 0 {
				textAlign := datadog.WidgetTextAlign(v)
				definition.TextAlign = &textAlign
			}
			if v, ok := widgetDefinition["show_tick"]; ok {
				definition.ShowTick = datadog.Bool(v.(bool))
			}
			if v, ok := widgetDefinition["tick_pos"].(string); ok && len(v) != 0 {
				definition.TickPos = datadog.String(v)
			}
			if v, ok := widgetDefinition["tick_edge"].(string); ok && len(v) != 0 {
				definition.TickEdge = datadog.String(v)
			}

			d.Definition = definition
		default:
			return nil, fmt.Errorf("Invalid widget type: %s", *widgetType)

		}
	}
	return &datadogWidgets, nil
}

func buildDashboard(d *schema.ResourceData) (*datadog.Board, error) {
	layoutType := datadog.String(d.Get("layout_type").(string))
	terraformWidgets := d.Get("widget").([]interface{})
	widgets, err := buildDashboardWidgets(&terraformWidgets, *layoutType)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse widgets: %s", err.Error())
	}
	dashboard := datadog.Board{
		Id:         datadog.String(d.Id()),
		Title:      datadog.String(d.Get("title").(string)),
		Widgets:    *widgets,
		LayoutType: layoutType,
	}

	if v, ok := d.GetOk("description"); ok {
		dashboard.Description = datadog.String(v.(string))
	}
	if v, ok := d.GetOk("is_read_only"); ok {
		dashboard.IsReadOnly = datadog.Bool(v.(bool))
	}
	if v, ok := d.GetOk("notify_list"); ok {
		notifyList := []string{}
		for _, s := range v.([]interface{}) {
			notifyList = append(notifyList, s.(string))
		}
		dashboard.NotifyList = notifyList
	}
	if v, ok := d.GetOk("template_variable"); ok {
		templateVariables := v.([]interface{})
		dashboard.TemplateVariables = *buildTemplateVariables(&templateVariables)
	}
	return &dashboard, nil
}

func resourceDatadogDashboardCreate(d *schema.ResourceData, meta interface{}) error {
	dashboard, err := buildDashboard(d)
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
	dashboard, err := buildDashboard(d)
	if err != nil {
		return fmt.Errorf("Failed to parse resource configuration: %s", err.Error())
	}
	if err = meta.(*datadog.Client).UpdateBoard(dashboard); err != nil {
		return fmt.Errorf("Failed to update dashboard using Datadog API: %s", err.Error())
	}
	return resourceDatadogDashboardRead(d, meta)
}

func buildTerraformWidget(datadogWidget datadog.BoardWidget) map[string]interface{} {
	widgetMap := map[string]interface{}{}
	definitionMap := map[string]interface{}{}

	if datadogWidget.Id != nil {
		widgetMap["id"] = *datadogWidget.Id
	}
	if layout := datadogWidget.Layout; layout != nil {
		layoutMap := map[string]string{}
		layoutMap["x"] = strconv.FormatFloat(*layout.X, 'f', -1, 64)
		layoutMap["y"] = strconv.FormatFloat(*layout.Y, 'f', -1, 64)
		layoutMap["height"] = strconv.FormatFloat(*layout.Height, 'f', -1, 64)
		layoutMap["width"] = strconv.FormatFloat(*layout.Width, 'f', -1, 64)
		widgetMap["layout"] = layoutMap
	}

	// We try to determine the widget type using two methods:
	var widgetType string
	// If this is a group widget, the definition will be a map
	if v, ok := datadogWidget.Definition.(map[string]interface{}); ok {
		widgetType = v["type"].(string)
	} else {
		// If this is a root widget, determine the widget type based on the definition type
		switch datadogWidget.Definition.(type) {
		case datadog.GroupDefinition:
			widgetType = *datadogWidget.Definition.(datadog.GroupDefinition).Type
		case datadog.NoteDefinition:
			widgetType = *datadogWidget.Definition.(datadog.NoteDefinition).Type
		}
	}

	// Regardless of how we determine the type, the conversion is the same
	switch widgetType {
	case "group":
		definition := datadogWidget.Definition.(datadog.GroupDefinition)
		// Required params
		definitionMap["type"] = widgetType

		groupWidgets := []map[string]interface{}{}
		for _, groupWidget := range definition.Widgets {
			groupWidgets = append(groupWidgets, buildTerraformWidget(groupWidget))
		}
		definitionMap["widget"] = groupWidgets

		// Optional params
		if definition.Title != nil {
			definitionMap["title"] = *definition.Title
		}

	case "note":
		definition := datadogWidget.Definition.(datadog.NoteDefinition)
		// Required params
		definitionMap["type"] = widgetType
		definitionMap["content"] = *definition.Content
		// Optional params
		if definition.BackgroundColor != nil {
			definitionMap["background_color"] = *definition.BackgroundColor
		}
		if definition.FontSize != nil {
			definitionMap["font_size"] = *definition.FontSize
		}
		if definition.TextAlign != nil {
			definitionMap["text_align"] = *definition.TextAlign
		}
		if definition.ShowTick != nil {
			definitionMap["show_tick"] = *definition.ShowTick
		}
		if definition.TickPos != nil {
			definitionMap["tick_pos"] = *definition.TickPos
		}
		if definition.TickEdge != nil {
			definitionMap["tick_edge"] = *definition.TickEdge
		}
	default:
		// do nothing
	}

	// The definition is defined in the schema as a TypeList with a MaxItems of 1
	// So we we need to convert it to an array
	// This is in order to allow nested structure
	definition := []map[string]interface{}{}
	definition = append(definition, definitionMap)
	widgetMap["definition"] = definition
	return widgetMap
}

func resourceDatadogDashboardRead(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()
	dashboard, err := meta.(*datadog.Client).GetBoard(id)
	if err != nil {
		return err
	}
	log.Printf("[DataDog] dashboard: %v", pretty.Sprint(dashboard))
	if err := d.Set("title", dashboard.Title); err != nil {
		return err
	}

	widgets := []map[string]interface{}{}
	for _, datadogWidget := range dashboard.Widgets {
		widgets = append(widgets, buildTerraformWidget(datadogWidget))
	}
	log.Printf("[DataDog] widgets: %v", pretty.Sprint(widgets))
	if err := d.Set("widget", widgets); err != nil {
		return err
	}

	if err := d.Set("layout_type", dashboard.LayoutType); err != nil {
		return err
	}
	if err := d.Set("description", dashboard.Description); err != nil {
		return err
	}
	if err := d.Set("is_read_only", dashboard.IsReadOnly); err != nil {
		return err
	}

	notifyList := []string{}
	for _, notifyListItem := range dashboard.NotifyList {
		notifyList = append(notifyList, notifyListItem)
	}
	if err := d.Set("notify_list", notifyList); err != nil {
		return err
	}

	templateVariables := []map[string]string{}
	for _, templateVariable := range dashboard.TemplateVariables {
		tv := map[string]string{}
		if v, ok := templateVariable.GetNameOk(); ok {
			tv["name"] = v
		}
		if v, ok := templateVariable.GetPrefixOk(); ok {
			tv["prefix"] = v
		}
		if v, ok := templateVariable.GetDefaultOk(); ok {
			tv["default"] = v
		}
		templateVariables = append(templateVariables, tv)
	}
	if err := d.Set("template_variable", templateVariables); err != nil {
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

// Validation functions
func validateDashboardLayoutType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	switch value {
	case "ordered", "free":
		break
	default:
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid layout_type parameter %q. Valid parameters are 'ordered' or 'free' ", k, value))
	}
	return
}

func validateGroupWidgetLayoutType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	switch value {
	case "ordered":
		break
	default:
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid layout_type parameter %q. Valid parameter is 'ordered' ", k, value))
	}
	return
}
