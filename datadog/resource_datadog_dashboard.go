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

// Float64 is a helper routine that allocates a new float value
// to store v and returns a pointer to it.
// TODO: move to go-datadog-api/helpers.go
func Float64(v float64) *float64 { return &v }

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
	widget := &schema.Schema{
		Type:        schema.TypeList,
		Required:    true,
		Description: "A list of graph definitions.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:        schema.TypeInt,
					Optional:    true,
					Description: "The id of the widget.",
				},
				"layout": widgetLayout,
				"definition": {
					Type:     schema.TypeMap,
					Optional: true,
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
				Type:        schema.TypeString,
				Required:    true,
				Description: "The layout type of the dashboard.'free' or 'ordered'",
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
		widgetDefinition := widgetMap["definition"].(map[string]interface{})
		widgetType := datadog.String(widgetDefinition["type"].(string))

		datadogWidgets[i] = datadog.BoardWidget{}
		d := &datadogWidgets[i]

		switch *widgetType {
		case "note":
			definition := &datadog.NoteDefinition{}

			// Required params
			definition.Type = widgetType
			if v, ok := widgetDefinition["content"]; ok {
				definition.Content = datadog.String(v.(string))
			}
			// Optional params
			if v, ok := widgetDefinition["background_color"]; ok {
				definition.BackgroundColor = datadog.String(v.(string))
			}
			if v, ok := widgetDefinition["font_size"]; ok {
				definition.FontSize = datadog.String(v.(string))
			}
			if v, ok := widgetDefinition["text_align"]; ok {
				textAlign := datadog.WidgetTextAlign(v.(string))
				definition.TextAlign = &textAlign
			}
			if v, ok := widgetDefinition["show_tick"]; ok {
				v, _ = strconv.ParseBool(v.(string))
				definition.ShowTick = datadog.Bool(v.(bool))
			}
			if v, ok := widgetDefinition["tick_pos"]; ok {
				definition.TickPos = datadog.String(v.(string))
			}
			if v, ok := widgetDefinition["tick_edge"]; ok {
				definition.TickEdge = datadog.String(v.(string))
			}
			d.Definition = definition
		default:
			return nil, fmt.Errorf("Invalid widget type: %s", *widgetType)

		}

		if layoutType == "free" {
			layoutMap := widgetMap["layout"].(map[string]interface{})
			layout := &datadog.WidgetLayout{}
			if v, err := strconv.ParseFloat(layoutMap["x"].(string), 64); err == nil {
				layout.X = Float64(v)
			}
			if v, err := strconv.ParseFloat(layoutMap["y"].(string), 64); err == nil {
				layout.Y = Float64(v)
			}
			if v, err := strconv.ParseFloat(layoutMap["height"].(string), 64); err == nil {
				layout.Height = Float64(v)
			}
			if v, err := strconv.ParseFloat(layoutMap["width"].(string), 64); err == nil {
				layout.Width = Float64(v)
			}

			d.Layout = layout

		}

	}
	return &datadogWidgets, nil
}

func buildDashboard(d *schema.ResourceData) (*datadog.Board, error) {
	layoutType := datadog.String(d.Get("layout_type").(string))
	terraformWidgets := d.Get("widget").([]interface{})
	terraformTemplateVariables := d.Get("template_variable").([]interface{})
	widgets, err := buildDashboardWidgets(&terraformWidgets, *layoutType)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse widgets: %s", err.Error())
	}
	dashboard := datadog.Board{
		// Id:                datadog.String(d.Id()),
		Title:             datadog.String(d.Get("title").(string)),
		Widgets:           *widgets,
		LayoutType:        layoutType,
		Description:       datadog.String(d.Get("description").(string)),
		IsReadOnly:        datadog.Bool(d.Get("is_read_only").(bool)),
		TemplateVariables: *buildTemplateVariables(&terraformTemplateVariables),
		// AuthorHandle:      datadog.String(d.Get("author_handle").(string)),
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
	ddefinitionMap := map[string]interface{}{}

	if datadogWidget.Id != nil {
		widgetMap["id"] = *datadogWidget.Id
	}
	if layout := datadogWidget.Layout; layout != nil {
		layoutMap := map[string]string{}
		layoutMap["x"] = fmt.Sprintf("%f", *layout.X)
		layoutMap["y"] = fmt.Sprintf("%f", *layout.Y)
		layoutMap["height"] = fmt.Sprintf("%f", *layout.Height)
		layoutMap["width"] = fmt.Sprintf("%f", *layout.Width)
		widgetMap["layout"] = layoutMap
	}

	switch datadogWidget.Definition.(type) {
	case datadog.NoteDefinition:
		definition := datadogWidget.Definition.(datadog.NoteDefinition)
		// Required params
		ddefinitionMap["type"] = *definition.Type
		ddefinitionMap["contente"] = *definition.Content
		// Optional params
		if definition.BackgroundColor != nil {
			ddefinitionMap["background_color"] = *definition.BackgroundColor
		}
		if definition.FontSize != nil {
			ddefinitionMap["font_size"] = *definition.FontSize
		}
		if definition.TextAlign != nil {
			ddefinitionMap["text_align"] = *definition.TextAlign
		}
		if definition.ShowTick != nil {
			ddefinitionMap["show_tick"] = strconv.FormatBool(*definition.ShowTick)
		}
		if definition.TickPos != nil {
			ddefinitionMap["tick_pos"] = *definition.TickPos
		}
		if definition.TickEdge != nil {
			ddefinitionMap["tick_edge"] = *definition.TickEdge
		}
	default:
		// return "", errors.New("unsupported id type")
	}

	widgetMap["definition"] = ddefinitionMap
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
	if err := d.Set("layout_type", dashboard.LayoutType); err != nil {
		return err
	}
	if err := d.Set("description", dashboard.Description); err != nil {
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
