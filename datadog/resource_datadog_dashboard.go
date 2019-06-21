package datadog

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
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
			"widget": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "The list of widgets to display on the dashboard.",
				Elem: &schema.Resource{
					Schema: getWidgetSchema(),
				},
			},
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

	if err = d.Set("title", dashboard.GetTitle()); err != nil {
		return err
	}
	if err = d.Set("layout_type", dashboard.GetLayoutType()); err != nil {
		return err
	}
	if err = d.Set("description", dashboard.GetDescription()); err != nil {
		return err
	}
	if err = d.Set("is_read_only", dashboard.GetIsReadOnly()); err != nil {
		return err
	}

	// Set widgets
	terraformWidgets, err := buildTerraformWidgets(&dashboard.Widgets)
	if err != nil {
		return err
	}
	if err := d.Set("widget", terraformWidgets); err != nil {
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

func buildDatadogDashboard(d *schema.ResourceData) (*datadog.Board, error) {
	var dashboard datadog.Board

	dashboard.SetId(d.Id())

	if v, ok := d.GetOk("title"); ok {
		dashboard.SetTitle(v.(string))
	}
	if v, ok := d.GetOk("layout_type"); ok {
		dashboard.SetLayoutType(v.(string))
	}
	if v, ok := d.GetOk("description"); ok {
		dashboard.SetDescription(v.(string))
	}
	if v, ok := d.GetOk("is_read_only"); ok {
		dashboard.SetIsReadOnly(v.(bool))
	}

	// Build Widgets
	terraformWidgets := d.Get("widget").([]interface{})
	datadogWidgets, err := buildDatadogWidgets(&terraformWidgets)
	if err != nil {
		return nil, err
	}
	dashboard.Widgets = *datadogWidgets

	// Build NotifyList
	notifyList := d.Get("notify_list").([]interface{})
	dashboard.NotifyList = *buildDatadogNotifyList(&notifyList)

	// Build TemplateVariables
	templateVariables := d.Get("template_variable").([]interface{})
	dashboard.TemplateVariables = *buildDatadogTemplateVariables(&templateVariables)

	return &dashboard, nil
}

//
// Template Variable helpers
//

func getTemplateVariableSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
	}
}

func buildDatadogTemplateVariables(terraformTemplateVariables *[]interface{}) *[]datadog.TemplateVariable {
	datadogTemplateVariables := make([]datadog.TemplateVariable, len(*terraformTemplateVariables))
	for i, _terraformTemplateVariable := range *terraformTemplateVariables {
		terraformTemplateVariable := _terraformTemplateVariable.(map[string]interface{})
		var datadogTemplateVariable datadog.TemplateVariable
		if v, ok := terraformTemplateVariable["name"].(string); ok && len(v) != 0 {
			datadogTemplateVariable.SetName(v)
		}
		if v, ok := terraformTemplateVariable["prefix"].(string); ok && len(v) != 0 {
			datadogTemplateVariable.SetPrefix(v)
		}
		if v, ok := terraformTemplateVariable["default"].(string); ok && len(v) != 0 {
			datadogTemplateVariable.SetDefault(v)
		}
		datadogTemplateVariables[i] = datadogTemplateVariable
	}
	return &datadogTemplateVariables
}

func buildTerraformTemplateVariables(datadogTemplateVariables *[]datadog.TemplateVariable) *[]map[string]string {
	terraformTemplateVariables := make([]map[string]string, len(*datadogTemplateVariables))
	for i, templateVariable := range *datadogTemplateVariables {
		terraformTemplateVariable := map[string]string{}
		if v, ok := templateVariable.GetNameOk(); ok {
			terraformTemplateVariable["name"] = v
		}
		if v, ok := templateVariable.GetPrefixOk(); ok {
			terraformTemplateVariable["prefix"] = v
		}
		if v, ok := templateVariable.GetDefaultOk(); ok {
			terraformTemplateVariable["default"] = v
		}
		terraformTemplateVariables[i] = terraformTemplateVariable
	}
	return &terraformTemplateVariables
}

//
// Notify List helpers
//

func buildDatadogNotifyList(terraformNotifyList *[]interface{}) *[]string {
	datadogNotifyList := make([]string, len(*terraformNotifyList))
	for i, authorHandle := range *terraformNotifyList {
		datadogNotifyList[i] = authorHandle.(string)
	}
	return &datadogNotifyList
}

func buildTerraformNotifyList(datadogNotifyList *[]string) *[]string {
	terraformNotifyList := make([]string, len(*datadogNotifyList))
	for i, authorHandle := range *datadogNotifyList {
		terraformNotifyList[i] = authorHandle
	}
	return &terraformNotifyList
}

//
// Widget helpers
//

// The generic widget schema is a combination of the schema for a non-group widget
// and the schema for a Group Widget (which can contains only non-group widgets)
func getWidgetSchema() map[string]*schema.Schema {
	widgetSchema := getNonGroupWidgetSchema()
	widgetSchema["group_definition"] = &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "The definition for a Group widget",
		Elem: &schema.Resource{
			Schema: getGroupDefinitionSchema(),
		},
	}
	return widgetSchema
}

func getNonGroupWidgetSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"layout": {
			Type:        schema.TypeMap,
			Optional:    true,
			Description: "The layout of the widget on a 'free' dashboard",
			Elem: &schema.Resource{
				Schema: getWidgetLayoutSchema(),
			},
		},
		// A widget should implement exactly one of the following definitions
		// TODO: add a dynamic ConflictsWith to each type to enforece ^
		"note_definition": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The definition for a Note widget",
			Elem: &schema.Resource{
				Schema: getNoteDefinitionSchema(),
			},
		},
		"alert_graph_definition": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The definition for a Alert Graph widget",
			Elem: &schema.Resource{
				Schema: getAlertGraphDefinitionSchema(),
			},
		},
		"alert_value_definition": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The definition for a Alert Value widget",
			Elem: &schema.Resource{
				Schema: getAlertValueDefinitionSchema(),
			},
		},
		"check_status_definition": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The definition for a Check Status widget",
			Elem: &schema.Resource{
				Schema: getCheckStatusDefinitionSchema(),
			},
		},
		// "event_stream_definition": {
		// 	Type:        schema.TypeList,
		// 	Optional:    true,
		// 	MaxItems:    1,
		// 	Description: "The definition for a Check Status widget",
		// 	Elem: &schema.Resource{
		// 		Schema: getEventStreamDefinitionSchema(),
		// 	},
		// },
		"free_text_definition": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The definition for a Free Text widget",
			Elem: &schema.Resource{
				Schema: getFreeTextDefinitionSchema(),
			},
		},
		"iframe_definition": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The definition for an Iframe widget",
			Elem: &schema.Resource{
				Schema: getIframeDefinitionSchema(),
			},
		},
		"image_definition": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The definition for an Image widget",
			Elem: &schema.Resource{
				Schema: getImageDefinitionSchema(),
			},
		},
		"log_stream_definition": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The definition for an Log Stream widget",
			Elem: &schema.Resource{
				Schema: getLogStreamDefinitionSchema(),
			},
		},
		"timeseries_definition": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The definition for a Timeseries widget",
			Elem: &schema.Resource{
				Schema: getTimeseriesDefinitionSchema(),
			},
		},
	}
}

func buildDatadogWidgets(terraformWidgets *[]interface{}) (*[]datadog.BoardWidget, error) {
	datadogWidgets := make([]datadog.BoardWidget, len(*terraformWidgets))
	for i, terraformWidget := range *terraformWidgets {
		datadogWidget, err := buildDatadogWidget(terraformWidget.(map[string]interface{}))
		if err != nil {
			return nil, err
		}
		datadogWidgets[i] = *datadogWidget
	}
	return &datadogWidgets, nil
}

// Helper to build a Datadog widget from a Terraform widget
func buildDatadogWidget(terraformWidget map[string]interface{}) (*datadog.BoardWidget, error) {
	datadogWidget := datadog.BoardWidget{}

	// Build widget layout
	if v, ok := terraformWidget["layout"].(map[string]interface{}); ok && len(v) != 0 {
		datadogWidget.SetLayout(buildDatadogWidgetLayout(v))
	}

	// Build widget Definition
	if _def, ok := terraformWidget["group_definition"].([]interface{}); ok && len(_def) > 0 {
		if groupDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogDefinition, err := buildDatadogGroupDefinition(groupDefinition)
			if err != nil {
				return nil, err
			}
			datadogWidget.Definition = datadogDefinition
		}
	} else if _def, ok := terraformWidget["note_definition"].([]interface{}); ok && len(_def) > 0 {
		if noteDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogWidget.Definition = buildDatadogNoteDefinition(noteDefinition)
		}
	} else if _def, ok := terraformWidget["alert_graph_definition"].([]interface{}); ok && len(_def) > 0 {
		if alertGraphDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogWidget.Definition = buildDatadogAlertGraphDefinition(alertGraphDefinition)
		}
	} else if _def, ok := terraformWidget["alert_value_definition"].([]interface{}); ok && len(_def) > 0 {
		if alertValueDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogWidget.Definition = buildDatadogAlertValueDefinition(alertValueDefinition)
		}
	} else if _def, ok := terraformWidget["check_status_definition"].([]interface{}); ok && len(_def) > 0 {
		if checkStatusDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogWidget.Definition = buildDatadogCheckStatusDefinition(checkStatusDefinition)
		}
		// } else if _def, ok := terraformWidget["event_stream_definition"].([]interface{}); ok && len(_def) > 0 {
		// 	if eventStreamDefinition, ok := _def[0].(map[string]interface{}); ok {
		// 		datadogWidget.Definition = buildDatadogEventStreamDefinition(eventStreamDefinition)
		// 	}
	} else if _def, ok := terraformWidget["free_text_definition"].([]interface{}); ok && len(_def) > 0 {
		if freeTextDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogWidget.Definition = buildDatadogFreeTextDefinition(freeTextDefinition)
		}
	} else if _def, ok := terraformWidget["iframe_definition"].([]interface{}); ok && len(_def) > 0 {
		if iframeDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogWidget.Definition = buildDatadogIframeDefinition(iframeDefinition)
		}
	} else if _def, ok := terraformWidget["image_definition"].([]interface{}); ok && len(_def) > 0 {
		if imageDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogWidget.Definition = buildDatadogImageDefinition(imageDefinition)
		}
	} else if _def, ok := terraformWidget["log_stream_definition"].([]interface{}); ok && len(_def) > 0 {
		if logStreamDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogWidget.Definition = buildDatadogLogStreamDefinition(logStreamDefinition)
		}
	} else if _def, ok := terraformWidget["timeseries_definition"].([]interface{}); ok && len(_def) > 0 {
		if timeseriesDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogWidget.Definition = buildDatadogTimeseriesDefinition(timeseriesDefinition)
		}
	} else {
		return nil, fmt.Errorf("Failed to find valid definition in widget configuration")
	}

	return &datadogWidget, nil
}

// Helper to build a list of Terraform widgets from a list of Datadog widgets
func buildTerraformWidgets(datadogWidgets *[]datadog.BoardWidget) (*[]map[string]interface{}, error) {
	terraformWidgets := make([]map[string]interface{}, len(*datadogWidgets))
	for i, datadogWidget := range *datadogWidgets {
		terraformWidget, err := buildTerraformWidget(datadogWidget)
		if err != nil {
			return nil, err
		}
		terraformWidgets[i] = terraformWidget
	}
	return &terraformWidgets, nil
}

// Helper to build a Terraform widget from a Datadog widget
func buildTerraformWidget(datadogWidget datadog.BoardWidget) (map[string]interface{}, error) {
	terraformWidget := map[string]interface{}{}

	// Build layout
	if datadogWidget.Layout != nil {
		terraformWidget["layout"] = buildTerraformWidgetLayout(*datadogWidget.Layout)
	}

	// Build definition
	widgetType, err := datadogWidget.GetWidgetType()
	if err != nil {
		return nil, err
	}
	switch widgetType {
	case datadog.GROUP_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.GroupDefinition)
		terraformDefinition := buildTerraformGroupDefinition(datadogDefinition)
		terraformWidget["group_definition"] = []map[string]interface{}{terraformDefinition}
	case datadog.NOTE_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.NoteDefinition)
		terraformDefinition := buildTerraformNoteDefinition(datadogDefinition)
		terraformWidget["note_definition"] = []map[string]interface{}{terraformDefinition}
	case datadog.ALERT_GRAPH_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.AlertGraphDefinition)
		terraformDefinition := buildTerraformAlertGraphDefinition(datadogDefinition)
		terraformWidget["alert_graph_definition"] = []map[string]interface{}{terraformDefinition}
	case datadog.ALERT_VALUE_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.AlertValueDefinition)
		terraformDefinition := buildTerraformAlertValueDefinition(datadogDefinition)
		terraformWidget["alert_value_definition"] = []map[string]interface{}{terraformDefinition}
	case datadog.CHECK_STATUS_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.CheckStatusDefinition)
		terraformDefinition := buildTerraformCheckStatusDefinition(datadogDefinition)
		terraformWidget["check_status_definition"] = []map[string]interface{}{terraformDefinition}
	// case datadog.EVENT_STREAM_WIDGET:
	// 	datadogDefinition := datadogWidget.Definition.(datadog.EventStreamDefinition)
	// 	terraformDefinition := buildTerraformEventStreamDefinition(datadogDefinition)
	// 	terraformWidget["event_stream_definition"] = []map[string]interface{}{terraformDefinition}
	case datadog.FREE_TEXT_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.FreeTextDefinition)
		terraformDefinition := buildTerraformFreeTextDefinition(datadogDefinition)
		terraformWidget["free_text_definition"] = []map[string]interface{}{terraformDefinition}
	case datadog.IFRAME_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.IframeDefinition)
		terraformDefinition := buildTerraformIframeDefinition(datadogDefinition)
		terraformWidget["iframe_definition"] = []map[string]interface{}{terraformDefinition}
	case datadog.IMAGE_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.ImageDefinition)
		terraformDefinition := buildTerraformImageDefinition(datadogDefinition)
		terraformWidget["image_definition"] = []map[string]interface{}{terraformDefinition}
	case datadog.LOG_STREAM_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.LogStreamDefinition)
		terraformDefinition := buildTerraformLogStreamDefinition(datadogDefinition)
		terraformWidget["log_stream_definition"] = []map[string]interface{}{terraformDefinition}
	case datadog.TIMESERIES_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.TimeseriesDefinition)
		terraformDefinition := buildTerraformTimeseriesDefinition(datadogDefinition)
		terraformWidget["timeseries_definition"] = []map[string]interface{}{terraformDefinition}
	default:
		return nil, fmt.Errorf("Unsupported widget type: %s - %s", widgetType, datadog.TIMESERIES_WIDGET)
	}

	return terraformWidget, nil
}

//
// Widget Layout helpers
//

func getWidgetLayoutSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
	}
}

func buildDatadogWidgetLayout(terraformLayout map[string]interface{}) datadog.WidgetLayout {
	datadogLayout := datadog.WidgetLayout{}

	if _v, ok := terraformLayout["x"].(string); ok && len(_v) != 0 {
		if v, err := strconv.ParseFloat(_v, 64); err == nil {
			datadogLayout.SetX(v)
		}
	}
	if _v, ok := terraformLayout["y"].(string); ok && len(_v) != 0 {
		if v, err := strconv.ParseFloat(_v, 64); err == nil {
			datadogLayout.SetY(v)
		}
	}
	if _v, ok := terraformLayout["height"].(string); ok && len(_v) != 0 {
		if v, err := strconv.ParseFloat(_v, 64); err == nil {
			datadogLayout.SetHeight(v)
		}
	}
	if _v, ok := terraformLayout["width"].(string); ok && len(_v) != 0 {
		if v, err := strconv.ParseFloat(_v, 64); err == nil {
			datadogLayout.SetWidth(v)
		}
	}
	return datadogLayout
}

func buildTerraformWidgetLayout(datadogLayout datadog.WidgetLayout) map[string]string {
	terraformLayout := map[string]string{}

	if v, ok := datadogLayout.GetXOk(); ok {
		terraformLayout["x"] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	if v, ok := datadogLayout.GetYOk(); ok {
		terraformLayout["y"] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	if v, ok := datadogLayout.GetHeightOk(); ok {
		terraformLayout["height"] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	if v, ok := datadogLayout.GetWidthOk(); ok {
		terraformLayout["width"] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	return terraformLayout
}

//
// Group Widget helpers
//

func getGroupDefinitionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"widget": {
			Type:        schema.TypeList,
			Required:    true,
			Description: "The list of widgets in this group.",
			Elem: &schema.Resource{
				Schema: getNonGroupWidgetSchema(),
			},
		},
		"layout_type": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The layout type of the group, only 'ordered' for now.",
		},
		"title": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The title of the group.",
		},
	}
}

func buildDatadogGroupDefinition(terraformGroupDefinition map[string]interface{}) (*datadog.GroupDefinition, error) {
	datadogGroupDefinition := datadog.GroupDefinition{}
	datadogGroupDefinition.SetType(datadog.GROUP_WIDGET)

	if v, ok := terraformGroupDefinition["widget"].([]interface{}); ok && len(v) != 0 {
		datadogWidgets, err := buildDatadogWidgets(&v)
		if err != nil {
			return nil, err
		}
		datadogGroupDefinition.Widgets = *datadogWidgets
	}
	if v, ok := terraformGroupDefinition["layout_type"].(string); ok && len(v) != 0 {
		datadogGroupDefinition.SetLayoutType(v)
	}
	if v, ok := terraformGroupDefinition["title"].(string); ok && len(v) != 0 {
		datadogGroupDefinition.SetTitle(v)
	}

	return &datadogGroupDefinition, nil
}

func buildTerraformGroupDefinition(datadogGroupDefinition datadog.GroupDefinition) map[string]interface{} {
	terraformGroupDefinition := map[string]interface{}{}

	groupWidgets := []map[string]interface{}{}
	for _, datadogGroupWidgets := range datadogGroupDefinition.Widgets {
		newGroupWidget, _ := buildTerraformWidget(datadogGroupWidgets)
		groupWidgets = append(groupWidgets, newGroupWidget)
	}
	terraformGroupDefinition["widget"] = groupWidgets

	if v, ok := datadogGroupDefinition.GetLayoutTypeOk(); ok {
		terraformGroupDefinition["layout_type"] = v
	}
	if v, ok := datadogGroupDefinition.GetTitleOk(); ok {
		terraformGroupDefinition["title"] = v
	}

	return terraformGroupDefinition
}

//
// Note Widget Definition helpers
//

func getNoteDefinitionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"content": {
			Type:     schema.TypeString,
			Required: true,
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
}

func buildDatadogNoteDefinition(terraformDefinition map[string]interface{}) *datadog.NoteDefinition {
	datadogDefinition := &datadog.NoteDefinition{}
	// Required params
	datadogDefinition.Type = datadog.String(datadog.NOTE_WIDGET)
	datadogDefinition.Content = datadog.String(terraformDefinition["content"].(string))
	// Optional params
	if v, ok := terraformDefinition["background_color"].(string); ok && len(v) != 0 {
		datadogDefinition.BackgroundColor = datadog.String(v)
	}
	if v, ok := terraformDefinition["font_size"].(string); ok && len(v) != 0 {
		datadogDefinition.FontSize = datadog.String(v)
	}
	if v, ok := terraformDefinition["text_align"].(string); ok && len(v) != 0 {
		datadogDefinition.TextAlign = datadog.String(v)
	}
	if v, ok := terraformDefinition["show_tick"]; ok {
		datadogDefinition.ShowTick = datadog.Bool(v.(bool))
	}
	if v, ok := terraformDefinition["tick_pos"].(string); ok && len(v) != 0 {
		datadogDefinition.TickPos = datadog.String(v)
	}
	if v, ok := terraformDefinition["tick_edge"].(string); ok && len(v) != 0 {
		datadogDefinition.TickEdge = datadog.String(v)
	}
	return datadogDefinition
}

func buildTerraformNoteDefinition(datadogDefinition datadog.NoteDefinition) map[string]interface{} {
	terraformDefinition := map[string]interface{}{}
	// Required params
	terraformDefinition["content"] = *datadogDefinition.Content
	// Optional params
	if datadogDefinition.BackgroundColor != nil {
		terraformDefinition["background_color"] = *datadogDefinition.BackgroundColor
	}
	if datadogDefinition.FontSize != nil {
		terraformDefinition["font_size"] = *datadogDefinition.FontSize
	}
	if datadogDefinition.TextAlign != nil {
		terraformDefinition["text_align"] = *datadogDefinition.TextAlign
	}
	if datadogDefinition.ShowTick != nil {
		terraformDefinition["show_tick"] = *datadogDefinition.ShowTick
	}
	if datadogDefinition.TickPos != nil {
		terraformDefinition["tick_pos"] = *datadogDefinition.TickPos
	}
	if datadogDefinition.TickEdge != nil {
		terraformDefinition["tick_edge"] = *datadogDefinition.TickEdge
	}
	return terraformDefinition
}

//
// Alert Graph Widget Definition helpers
//

func getAlertGraphDefinitionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"alert_id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"viz_type": {
			Type:     schema.TypeString,
			Required: true,
		},
		"title": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title_size": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title_align": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"time": {
			Type:     schema.TypeMap,
			Optional: true,
			Elem: &schema.Resource{
				Schema: getWidgetTimeSchema(),
			},
		},
	}
}

func buildDatadogAlertGraphDefinition(terraformDefinition map[string]interface{}) *datadog.AlertGraphDefinition {
	datadogDefinition := &datadog.AlertGraphDefinition{}
	// Required params
	datadogDefinition.Type = datadog.String(datadog.ALERT_GRAPH_WIDGET)
	datadogDefinition.AlertId = datadog.String(terraformDefinition["alert_id"].(string))
	datadogDefinition.VizType = datadog.String(terraformDefinition["viz_type"].(string))
	// Optional params
	if v, ok := terraformDefinition["title"].(string); ok && len(v) != 0 {
		datadogDefinition.Title = datadog.String(v)
	}
	if v, ok := terraformDefinition["title_size"].(string); ok && len(v) != 0 {
		datadogDefinition.TitleSize = datadog.String(v)
	}
	if v, ok := terraformDefinition["title_align"].(string); ok && len(v) != 0 {
		datadogDefinition.TitleAlign = datadog.String(v)
	}
	if v, ok := terraformDefinition["time"].(map[string]interface{}); ok && len(v) > 0 {
		datadogDefinition.Time = buildDatadogWidgetTime(v)
	}
	return datadogDefinition
}

func buildTerraformAlertGraphDefinition(datadogDefinition datadog.AlertGraphDefinition) map[string]interface{} {
	terraformDefinition := map[string]interface{}{}
	// Required params
	terraformDefinition["alert_id"] = *datadogDefinition.AlertId
	terraformDefinition["viz_type"] = *datadogDefinition.VizType
	// Optional params
	if datadogDefinition.Title != nil {
		terraformDefinition["title"] = *datadogDefinition.Title
	}
	if datadogDefinition.TitleSize != nil {
		terraformDefinition["title_size"] = *datadogDefinition.TitleSize
	}
	if datadogDefinition.TitleAlign != nil {
		terraformDefinition["title_align"] = *datadogDefinition.TitleAlign
	}
	if datadogDefinition.Time != nil {
		terraformDefinition["time"] = buildTerraformWidgetTime(*datadogDefinition.Time)
	}
	return terraformDefinition
}

//
// Alert Value Widget Definition helpers
//

func getAlertValueDefinitionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"alert_id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"precision": {
			Type:     schema.TypeInt,
			Optional: true,
		},
		"unit": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"text_align": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title_size": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title_align": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}
}

func buildDatadogAlertValueDefinition(terraformDefinition map[string]interface{}) *datadog.AlertValueDefinition {
	datadogDefinition := &datadog.AlertValueDefinition{}
	// Required params
	datadogDefinition.Type = datadog.String(datadog.ALERT_VALUE_WIDGET)
	datadogDefinition.AlertId = datadog.String(terraformDefinition["alert_id"].(string))
	// Optional params
	if v, ok := terraformDefinition["precision"].(int); ok && v != 0 {
		datadogDefinition.SetPrecision(v)
	}
	if v, ok := terraformDefinition["unit"].(string); ok && len(v) != 0 {
		datadogDefinition.SetUnit(v)
	}
	if v, ok := terraformDefinition["text_align"].(string); ok && len(v) != 0 {
		datadogDefinition.SetTextAlign(v)
	}
	if v, ok := terraformDefinition["title"].(string); ok && len(v) != 0 {
		datadogDefinition.SetTitle(v)
	}
	if v, ok := terraformDefinition["title_size"].(string); ok && len(v) != 0 {
		datadogDefinition.SetTitleSize(v)
	}
	if v, ok := terraformDefinition["title_align"].(string); ok && len(v) != 0 {
		datadogDefinition.SetTitleAlign(v)
	}
	return datadogDefinition
}

func buildTerraformAlertValueDefinition(datadogDefinition datadog.AlertValueDefinition) map[string]interface{} {
	terraformDefinition := map[string]interface{}{}
	// Required params
	terraformDefinition["alert_id"] = *datadogDefinition.AlertId
	// Optional params
	if datadogDefinition.Precision != nil {
		terraformDefinition["precision"] = *datadogDefinition.Precision
	}
	if datadogDefinition.Unit != nil {
		terraformDefinition["unit"] = *datadogDefinition.Unit
	}
	if datadogDefinition.TextAlign != nil {
		terraformDefinition["text_align"] = *datadogDefinition.TextAlign
	}
	if datadogDefinition.Title != nil {
		terraformDefinition["title"] = *datadogDefinition.Title
	}
	if datadogDefinition.TitleSize != nil {
		terraformDefinition["title_size"] = *datadogDefinition.TitleSize
	}
	if datadogDefinition.TitleAlign != nil {
		terraformDefinition["title_align"] = *datadogDefinition.TitleAlign
	}
	return terraformDefinition
}

//
// Event Stream Widget Definition helpers
//

func getEventStreamDefinitionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"query": {
			Type:     schema.TypeString,
			Required: true,
		},
		// "tags_execution": {
		// 	Type:     schema.TypeString,
		// 	Optional: true,
		// },
		"event_size": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title_size": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title_align": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"time": {
			Type:     schema.TypeMap,
			Optional: true,
			Elem: &schema.Resource{
				Schema: getWidgetTimeSchema(),
			},
		},
	}
}

func buildDatadogEventStreamDefinition(terraformDefinition map[string]interface{}) *datadog.EventStreamDefinition {
	datadogDefinition := &datadog.EventStreamDefinition{}
	// Required params
	datadogDefinition.Type = datadog.String(datadog.EVENT_STREAM_WIDGET)
	datadogDefinition.Query = datadog.String(terraformDefinition["query"].(string))
	// Optional params
	if v, ok := terraformDefinition["event_size"].(string); ok && len(v) != 0 {
		datadogDefinition.SetEventSize(v)
	}
	if v, ok := terraformDefinition["title"].(string); ok && len(v) != 0 {
		datadogDefinition.SetTitle(v)
	}
	if v, ok := terraformDefinition["title_size"].(string); ok && len(v) != 0 {
		datadogDefinition.SetTitleSize(v)
	}
	if v, ok := terraformDefinition["title_align"].(string); ok && len(v) != 0 {
		datadogDefinition.SetTitleAlign(v)
	}
	if v, ok := terraformDefinition["time"].(map[string]interface{}); ok && len(v) > 0 {
		datadogDefinition.SetTime(*buildDatadogWidgetTime(v))
	}
	return datadogDefinition
}

func buildTerraformEventStreamDefinition(datadogDefinition datadog.EventStreamDefinition) map[string]interface{} {
	terraformDefinition := map[string]interface{}{}
	// Required params
	terraformDefinition["query"] = *datadogDefinition.Query
	// Optional params
	if datadogDefinition.EventSize != nil {
		terraformDefinition["event_size"] = *datadogDefinition.EventSize
	}
	if datadogDefinition.Title != nil {
		terraformDefinition["title"] = *datadogDefinition.Title
	}
	if datadogDefinition.TitleSize != nil {
		terraformDefinition["title_size"] = *datadogDefinition.TitleSize
	}
	if datadogDefinition.TitleAlign != nil {
		terraformDefinition["title_align"] = *datadogDefinition.TitleAlign
	}
	if datadogDefinition.Time != nil {
		terraformDefinition["time"] = buildTerraformWidgetTime(*datadogDefinition.Time)
	}
	return terraformDefinition
}

//
// Check Status Widget Definition helpers
//

func getCheckStatusDefinitionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"check": {
			Type:     schema.TypeString,
			Required: true,
		},
		"grouping": {
			Type:     schema.TypeString,
			Required: true,
		},
		"group": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"group_by": {
			Type:     schema.TypeList,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"tags": {
			Type:     schema.TypeList,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"title": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title_size": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title_align": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"time": {
			Type:     schema.TypeMap,
			Optional: true,
			Elem: &schema.Resource{
				Schema: getWidgetTimeSchema(),
			},
		},
	}
}

func buildDatadogCheckStatusDefinition(terraformDefinition map[string]interface{}) *datadog.CheckStatusDefinition {
	datadogDefinition := &datadog.CheckStatusDefinition{}
	// Required params
	datadogDefinition.Type = datadog.String(datadog.CHECK_STATUS_WIDGET)
	datadogDefinition.Check = datadog.String(terraformDefinition["check"].(string))
	datadogDefinition.Grouping = datadog.String(terraformDefinition["grouping"].(string))
	// Optional params
	if v, ok := terraformDefinition["group"].(string); ok && len(v) != 0 {
		datadogDefinition.SetGroup(v)
	}
	if terraformGroupBys, ok := terraformDefinition["group_by"].([]interface{}); ok && len(terraformGroupBys) > 0 {
		datadogGroupBys := make([]string, len(terraformGroupBys))
		for i, groupBy := range terraformGroupBys {
			datadogGroupBys[i] = groupBy.(string)
		}
		datadogDefinition.GroupBy = datadogGroupBys
	}
	if terraformTags, ok := terraformDefinition["tags"].([]interface{}); ok && len(terraformTags) > 0 {
		datadogTags := make([]string, len(terraformTags))
		for i, tag := range terraformTags {
			datadogTags[i] = tag.(string)
		}
		datadogDefinition.Tags = datadogTags
	}
	if v, ok := terraformDefinition["title"].(string); ok && len(v) != 0 {
		datadogDefinition.SetTitle(v)
	}
	if v, ok := terraformDefinition["title_size"].(string); ok && len(v) != 0 {
		datadogDefinition.SetTitleSize(v)
	}
	if v, ok := terraformDefinition["title_align"].(string); ok && len(v) != 0 {
		datadogDefinition.SetTitleAlign(v)
	}
	if v, ok := terraformDefinition["time"].(map[string]interface{}); ok && len(v) > 0 {
		datadogDefinition.SetTime(*buildDatadogWidgetTime(v))
	}
	return datadogDefinition
}

func buildTerraformCheckStatusDefinition(datadogDefinition datadog.CheckStatusDefinition) map[string]interface{} {
	terraformDefinition := map[string]interface{}{}
	// Required params
	terraformDefinition["check"] = *datadogDefinition.Check
	terraformDefinition["grouping"] = *datadogDefinition.Grouping
	// Optional params
	if datadogDefinition.Group != nil {
		terraformDefinition["group"] = *datadogDefinition.Group
	}
	if datadogDefinition.GroupBy != nil {
		terraformGroupBys := make([]string, len(datadogDefinition.GroupBy))
		for i, datadogGroupBy := range datadogDefinition.GroupBy {
			terraformGroupBys[i] = datadogGroupBy
		}
		terraformDefinition["group_by"] = terraformGroupBys
	}
	if datadogDefinition.Tags != nil {
		terraformTags := make([]string, len(datadogDefinition.Tags))
		for i, datadogTag := range datadogDefinition.Tags {
			terraformTags[i] = datadogTag
		}
		terraformDefinition["tags"] = terraformTags
	}
	if datadogDefinition.Title != nil {
		terraformDefinition["title"] = *datadogDefinition.Title
	}
	if datadogDefinition.TitleSize != nil {
		terraformDefinition["title_size"] = *datadogDefinition.TitleSize
	}
	if datadogDefinition.TitleAlign != nil {
		terraformDefinition["title_align"] = *datadogDefinition.TitleAlign
	}
	if datadogDefinition.Time != nil {
		terraformDefinition["time"] = buildTerraformWidgetTime(*datadogDefinition.Time)
	}
	return terraformDefinition
}

//
// Free Text Definition helpers
//

func getFreeTextDefinitionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"text": {
			Type:     schema.TypeString,
			Required: true,
		},
		"color": {
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
	}
}

func buildDatadogFreeTextDefinition(terraformDefinition map[string]interface{}) *datadog.FreeTextDefinition {
	datadogDefinition := &datadog.FreeTextDefinition{}
	// Required params
	datadogDefinition.Type = datadog.String(datadog.FREE_TEXT_WIDGET)
	datadogDefinition.SetText(terraformDefinition["text"].(string))
	// Optional params
	if v, ok := terraformDefinition["color"].(string); ok && len(v) != 0 {
		datadogDefinition.SetColor(v)
	}
	if v, ok := terraformDefinition["font_size"].(string); ok && len(v) != 0 {
		datadogDefinition.SetFontSize(v)
	}
	if v, ok := terraformDefinition["text_align"].(string); ok && len(v) != 0 {
		datadogDefinition.SetTextAlign(v)
	}
	return datadogDefinition
}

func buildTerraformFreeTextDefinition(datadogDefinition datadog.FreeTextDefinition) map[string]interface{} {
	terraformDefinition := map[string]interface{}{}
	// Required params
	terraformDefinition["text"] = *datadogDefinition.Text
	// Optional params
	if datadogDefinition.Color != nil {
		terraformDefinition["color"] = *datadogDefinition.Color
	}
	if datadogDefinition.FontSize != nil {
		terraformDefinition["font_size"] = *datadogDefinition.FontSize
	}
	if datadogDefinition.TextAlign != nil {
		terraformDefinition["text_align"] = *datadogDefinition.TextAlign
	}
	return terraformDefinition
}

//
// Iframe Definition helpers
//

func getIframeDefinitionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"url": {
			Type:     schema.TypeString,
			Required: true,
		},
	}
}

func buildDatadogIframeDefinition(terraformDefinition map[string]interface{}) *datadog.IframeDefinition {
	datadogDefinition := &datadog.IframeDefinition{}
	// Required params
	datadogDefinition.Type = datadog.String(datadog.IFRAME_WIDGET)
	datadogDefinition.SetUrl(terraformDefinition["url"].(string))
	return datadogDefinition
}

func buildTerraformIframeDefinition(datadogDefinition datadog.IframeDefinition) map[string]interface{} {
	terraformDefinition := map[string]interface{}{}
	// Required params
	terraformDefinition["url"] = *datadogDefinition.Url
	return terraformDefinition
}

//
// Image Widget Definition helpers
//

func getImageDefinitionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"url": {
			Type:     schema.TypeString,
			Required: true,
		},
		"sizing": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"margin": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}
}

func buildDatadogImageDefinition(terraformDefinition map[string]interface{}) *datadog.ImageDefinition {
	datadogDefinition := &datadog.ImageDefinition{}
	// Required params
	datadogDefinition.Type = datadog.String(datadog.IMAGE_WIDGET)
	datadogDefinition.Url = datadog.String(terraformDefinition["url"].(string))
	// Optional params
	if v, ok := terraformDefinition["sizing"].(string); ok && len(v) != 0 {
		datadogDefinition.Sizing = datadog.String(v)
	}
	if v, ok := terraformDefinition["margin"].(string); ok && len(v) != 0 {
		datadogDefinition.Margin = datadog.String(v)
	}
	return datadogDefinition
}

func buildTerraformImageDefinition(datadogDefinition datadog.ImageDefinition) map[string]interface{} {
	terraformDefinition := map[string]interface{}{}
	// Required params
	terraformDefinition["url"] = *datadogDefinition.Url
	// Optional params
	if datadogDefinition.Sizing != nil {
		terraformDefinition["sizing"] = *datadogDefinition.Sizing
	}
	if datadogDefinition.Margin != nil {
		terraformDefinition["margin"] = *datadogDefinition.Margin
	}
	return terraformDefinition
}

//
// Log Stream Widget Definition helpers
//

func getLogStreamDefinitionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"logset": {
			Type:     schema.TypeString,
			Required: true,
		},
		"query": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"columns": {
			Type:     schema.TypeList,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"title": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title_size": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title_align": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"time": {
			Type:     schema.TypeMap,
			Optional: true,
			Elem: &schema.Resource{
				Schema: getWidgetTimeSchema(),
			},
		},
	}
}

func buildDatadogLogStreamDefinition(terraformDefinition map[string]interface{}) *datadog.LogStreamDefinition {
	datadogDefinition := &datadog.LogStreamDefinition{}
	// Required params
	datadogDefinition.Type = datadog.String(datadog.LOG_STREAM_WIDGET)
	datadogDefinition.Logset = datadog.String(terraformDefinition["logset"].(string))
	// Optional params
	if v, ok := terraformDefinition["query"].(string); ok && len(v) != 0 {
		datadogDefinition.Query = datadog.String(v)
	}
	if terraformColumns, ok := terraformDefinition["columns"].([]interface{}); ok && len(terraformColumns) > 0 {
		datadogColumns := make([]string, len(terraformColumns))
		for i, column := range terraformColumns {
			datadogColumns[i] = column.(string)
		}
		datadogDefinition.Columns = datadogColumns
	}
	if v, ok := terraformDefinition["title"].(string); ok && len(v) != 0 {
		datadogDefinition.Title = datadog.String(v)
	}
	if v, ok := terraformDefinition["title_size"].(string); ok && len(v) != 0 {
		datadogDefinition.TitleSize = datadog.String(v)
	}
	if v, ok := terraformDefinition["title_align"].(string); ok && len(v) != 0 {
		datadogDefinition.TitleAlign = datadog.String(v)
	}
	if v, ok := terraformDefinition["time"].(map[string]interface{}); ok && len(v) > 0 {
		datadogDefinition.Time = buildDatadogWidgetTime(v)
	}
	return datadogDefinition
}

func buildTerraformLogStreamDefinition(datadogDefinition datadog.LogStreamDefinition) map[string]interface{} {
	terraformDefinition := map[string]interface{}{}
	// Required params
	terraformDefinition["logset"] = *datadogDefinition.Logset
	// Optional params
	if datadogDefinition.Query != nil {
		terraformDefinition["query"] = *datadogDefinition.Query
	}
	if datadogDefinition.Columns != nil {
		terraformColumns := make([]string, len(datadogDefinition.Columns))
		for i, datadogColumn := range datadogDefinition.Columns {
			terraformColumns[i] = datadogColumn
		}
		terraformDefinition["columns"] = terraformColumns
	}
	if datadogDefinition.Title != nil {
		terraformDefinition["title"] = *datadogDefinition.Title
	}
	if datadogDefinition.TitleSize != nil {
		terraformDefinition["title_size"] = *datadogDefinition.TitleSize
	}
	if datadogDefinition.TitleAlign != nil {
		terraformDefinition["title_align"] = *datadogDefinition.TitleAlign
	}
	if datadogDefinition.Time != nil {
		terraformDefinition["time"] = buildTerraformWidgetTime(*datadogDefinition.Time)
	}
	return terraformDefinition
}

//
// Timeseries Widget Definition helpers
//

func getTimeseriesDefinitionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"request": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Resource{
				Schema: getTimeseriesRequestSchema(),
			},
		},
		"marker": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Resource{
				Schema: getWidgetMarkerSchema(),
			},
		},
		"title": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title_size": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"title_align": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"show_legend": {
			Type:     schema.TypeBool,
			Optional: true,
		},
		"legend_size": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"time": {
			Type:     schema.TypeMap,
			Optional: true,
			Elem: &schema.Resource{
				Schema: getWidgetTimeSchema(),
			},
		},
	}
}

func buildDatadogTimeseriesDefinition(terraformDefinition map[string]interface{}) *datadog.TimeseriesDefinition {
	datadogDefinition := &datadog.TimeseriesDefinition{}
	// Required params
	datadogDefinition.Type = datadog.String(datadog.TIMESERIES_WIDGET)
	terraformRequests := terraformDefinition["request"].([]interface{})
	datadogDefinition.Requests = *buildDatadogTimeseriesRequests(&terraformRequests)
	// Optional params
	if v, ok := terraformDefinition["marker"].([]interface{}); ok && len(v) > 0 {
		datadogDefinition.Markers = *buildDatadogWidgetMarkers(&v)
	}
	if v, ok := terraformDefinition["title"].(string); ok && len(v) != 0 {
		datadogDefinition.Title = datadog.String(v)
	}
	if v, ok := terraformDefinition["title_size"].(string); ok && len(v) != 0 {
		datadogDefinition.TitleSize = datadog.String(v)
	}
	if v, ok := terraformDefinition["title_align"].(string); ok && len(v) != 0 {
		datadogDefinition.TitleAlign = datadog.String(v)
	}
	if v, ok := terraformDefinition["time"].(map[string]interface{}); ok && len(v) > 0 {
		datadogDefinition.Time = buildDatadogWidgetTime(v)
	}
	return datadogDefinition
}

func buildTerraformTimeseriesDefinition(datadogDefinition datadog.TimeseriesDefinition) map[string]interface{} {
	terraformDefinition := map[string]interface{}{}
	// Required params
	terraformDefinition["request"] = buildTerraformTimeseriesRequests(&datadogDefinition.Requests)
	// Optional params
	if datadogDefinition.Markers != nil {
		terraformDefinition["marker"] = buildTerraformWidgetMarkers(&datadogDefinition.Markers)
	}
	if datadogDefinition.Title != nil {
		terraformDefinition["title"] = *datadogDefinition.Title
	}
	if datadogDefinition.TitleSize != nil {
		terraformDefinition["title_size"] = *datadogDefinition.TitleSize
	}
	if datadogDefinition.TitleAlign != nil {
		terraformDefinition["title_align"] = *datadogDefinition.TitleAlign
	}
	if datadogDefinition.Time != nil {
		terraformDefinition["time"] = buildTerraformWidgetTime(*datadogDefinition.Time)
	}
	return terraformDefinition
}

func getTimeseriesRequestSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		// A request should implement exactly one of the following type of query
		"q":             getMetricQuerySchema(),
		"apm_query":     getApmOrLogQuerySchema(),
		"log_query":     getApmOrLogQuerySchema(),
		"process_query": getProcessQuerySchema(),
		// Settings specific to Timeseries requests
		"display_type": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}
}
func buildDatadogTimeseriesRequests(terraformRequests *[]interface{}) *[]datadog.TimeseriesRequest {
	datadogRequests := make([]datadog.TimeseriesRequest, len(*terraformRequests))
	for i, _request := range *terraformRequests {
		terraformRequest := _request.(map[string]interface{})
		// Build TimeseriesRequest
		datadogTimeseriesRequest := datadog.TimeseriesRequest{}
		if v, ok := terraformRequest["q"].(string); ok && len(v) != 0 {
			datadogTimeseriesRequest.MetricQuery = datadog.String(v)
		} else if v, ok := terraformRequest["apm_query"].([]interface{}); ok && len(v) > 0 {
			apmQuery := v[0].(map[string]interface{})
			datadogTimeseriesRequest.ApmQuery = buildDatadogApmOrLogQuery(apmQuery)
		} else if v, ok := terraformRequest["log_query"].([]interface{}); ok && len(v) > 0 {
			logQuery := v[0].(map[string]interface{})
			datadogTimeseriesRequest.LogQuery = buildDatadogApmOrLogQuery(logQuery)
		} else if v, ok := terraformRequest["process_query"].([]interface{}); ok && len(v) > 0 {
			processQuery := v[0].(map[string]interface{})
			datadogTimeseriesRequest.ProcessQuery = buildDatadogProcessQuery(processQuery)
		}
		if v, ok := terraformRequest["display_type"].(string); ok && len(v) != 0 {
			datadogTimeseriesRequest.DisplayType = datadog.String(v)
		}
		datadogRequests[i] = datadogTimeseriesRequest
	}
	return &datadogRequests
}
func buildTerraformTimeseriesRequests(datadogTimeseriesRequests *[]datadog.TimeseriesRequest) *[]map[string]interface{} {
	terraformRequests := make([]map[string]interface{}, len(*datadogTimeseriesRequests))
	for i, datadogRequest := range *datadogTimeseriesRequests {
		terraformRequest := map[string]interface{}{}
		if datadogRequest.MetricQuery != nil {
			terraformRequest["q"] = *datadogRequest.MetricQuery
		} else if datadogRequest.ApmQuery != nil {
			terraformQuery := buildTerraformApmOrLogQuery(*datadogRequest.ApmQuery)
			terraformRequest["apm_query"] = []map[string]interface{}{terraformQuery}
		} else if datadogRequest.LogQuery != nil {
			terraformQuery := buildTerraformApmOrLogQuery(*datadogRequest.LogQuery)
			terraformRequest["log_query"] = []map[string]interface{}{terraformQuery}
		} else if datadogRequest.ProcessQuery != nil {
			terraformQuery := buildTerraformProcessQuery(*datadogRequest.ProcessQuery)
			terraformRequest["process_query"] = []map[string]interface{}{terraformQuery}
		}
		if datadogRequest.DisplayType != nil {
			terraformRequest["display_type"] = *datadogRequest.DisplayType
		}
		terraformRequests[i] = terraformRequest
	}
	return &terraformRequests
}

// Widget Time helpers
func getWidgetTimeSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"live_span": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}
}
func buildDatadogWidgetTime(terraformWidgetTime map[string]interface{}) *datadog.WidgetTime {
	datadogWidgetTime := &datadog.WidgetTime{}
	if v, ok := terraformWidgetTime["live_span"].(string); ok && len(v) != 0 {
		datadogWidgetTime.LiveSpan = datadog.String(v)
	}
	return datadogWidgetTime
}
func buildTerraformWidgetTime(datadogWidgetTime datadog.WidgetTime) map[string]string {
	terraformWidgetTime := map[string]string{}
	if datadogWidgetTime.LiveSpan != nil {
		terraformWidgetTime["live_span"] = *datadogWidgetTime.LiveSpan
	}
	return terraformWidgetTime
}

// Widget Marker helpers
func getWidgetMarkerSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"value": {
			Type:     schema.TypeString,
			Required: true,
		},
		"display_type": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"label": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}
}
func buildDatadogWidgetMarkers(terraformWidgetMarkers *[]interface{}) *[]datadog.WidgetMarker {
	datadogWidgetMarkers := make([]datadog.WidgetMarker, len(*terraformWidgetMarkers))
	for i, _marker := range *terraformWidgetMarkers {
		terraformMarker := _marker.(map[string]interface{})
		// Required
		datadogMarker := datadog.WidgetMarker{
			Value: datadog.String(terraformMarker["value"].(string)),
		}
		// Optional
		if v, ok := terraformMarker["display_type"].(string); ok && len(v) != 0 {
			datadogMarker.DisplayType = datadog.String(v)
		}
		if v, ok := terraformMarker["label"].(string); ok && len(v) != 0 {
			datadogMarker.Label = datadog.String(v)
		}
		datadogWidgetMarkers[i] = datadogMarker
	}
	return &datadogWidgetMarkers
}
func buildTerraformWidgetMarkers(datadogWidgetMarkers *[]datadog.WidgetMarker) *[]map[string]string {
	terraformWidgetMarkers := make([]map[string]string, len(*datadogWidgetMarkers))
	for i, datadogMarker := range *datadogWidgetMarkers {
		terraformMarker := map[string]string{}
		// Required params
		terraformMarker["value"] = *datadogMarker.Value
		// Optional params
		if datadogMarker.DisplayType != nil {
			terraformMarker["display_type"] = *datadogMarker.DisplayType
		}
		if datadogMarker.Label != nil {
			terraformMarker["label"] = *datadogMarker.Label
		}
		terraformWidgetMarkers[i] = terraformMarker
	}
	return &terraformWidgetMarkers
}

//
// Widget Query helpers
//

// Metric Query
func getMetricQuerySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	}
}

// APM or Log Query
func getApmOrLogQuerySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"index": {
					Type:     schema.TypeString,
					Required: true,
				},
				"compute": &schema.Schema{
					Type:     schema.TypeMap,
					Required: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"aggregation": {
								Type:     schema.TypeString,
								Required: true,
							},
							"facet": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"interval": {
								Type:     schema.TypeInt,
								Optional: true,
							},
						},
					},
				},
				"search": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"query": {
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
				"group_by": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"facet": {
								Type:     schema.TypeString,
								Required: true,
							},
							"limit": {
								Type:     schema.TypeInt,
								Optional: true,
							},
							"sort": {
								Type:     schema.TypeMap,
								Optional: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"aggregation": {
											Type:     schema.TypeString,
											Required: true,
										},
										"order": {
											Type:     schema.TypeString,
											Required: true,
										},
										"facet": {
											Type:     schema.TypeString,
											Optional: true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
func buildDatadogApmOrLogQuery(terraformQuery map[string]interface{}) *datadog.WidgetApmOrLogQuery {
	// Index
	datadogQuery := datadog.WidgetApmOrLogQuery{
		Index: datadog.String(terraformQuery["index"].(string)),
	}
	// Compute
	terraformCompute := terraformQuery["compute"].(map[string]interface{})
	datadogCompute := datadog.ApmOrLogQueryCompute{
		Aggregation: datadog.String(terraformCompute["aggregation"].(string)),
	}
	if v, ok := terraformCompute["facet"].(string); ok && len(v) != 0 {
		datadogCompute.Facet = datadog.String(v)
	}
	if v, err := strconv.ParseInt(terraformCompute["interval"].(string), 10, 64); err == nil {
		datadogCompute.Interval = datadog.Int(int(v))
	}
	datadogQuery.Compute = &datadogCompute
	// Search
	if terraformSearch, ok := terraformQuery["search"].(map[string]interface{}); ok && len(terraformSearch) > 0 {
		datadogQuery.Search = &datadog.ApmOrLogQuerySearch{
			Query: datadog.String(terraformSearch["query"].(string)),
		}
	}
	// GroupBy
	if terraformGroupBys, ok := terraformQuery["group_by"].([]interface{}); ok && len(terraformGroupBys) > 0 {
		datadogGroupBys := make([]datadog.ApmOrLogQueryGroupBy, len(terraformGroupBys))
		for i, _groupBy := range terraformGroupBys {
			groupBy := _groupBy.(map[string]interface{})
			// Facet
			datadogGroupBy := datadog.ApmOrLogQueryGroupBy{
				Facet: datadog.String(groupBy["facet"].(string)),
			}
			// Limit
			if v, ok := groupBy["limit"].(int); ok && v != 0 {
				datadogGroupBy.Limit = &v
			}
			// Sort
			if sort, ok := groupBy["sort"].(map[string]string); ok && len(sort) > 0 {
				datadogGroupBy.Sort = &datadog.ApmOrLogQueryGroupBySort{
					Aggregation: datadog.String(sort["aggregation"]),
					Order:       datadog.String(sort["order"]),
				}
				if len(sort["facet"]) > 0 {
					datadogGroupBy.Sort.Facet = datadog.String(sort["facet"])
				}
			}
			datadogGroupBys[i] = datadogGroupBy
		}
		datadogQuery.GroupBy = datadogGroupBys
	}
	return &datadogQuery
}
func buildTerraformApmOrLogQuery(datadogQuery datadog.WidgetApmOrLogQuery) map[string]interface{} {
	terraformQuery := map[string]interface{}{}
	// Index
	terraformQuery["index"] = *datadogQuery.Index
	// Compute
	terraformCompute := map[string]interface{}{
		"aggregation": *datadogQuery.Compute.Aggregation,
	}
	if datadogQuery.Compute.Facet != nil {
		terraformCompute["facet"] = *datadogQuery.Compute.Facet
	}
	if datadogQuery.Compute.Interval != nil {
		terraformCompute["interval"] = strconv.FormatInt(int64(*datadogQuery.Compute.Interval), 10)
	}
	terraformQuery["compute"] = terraformCompute
	// Search
	if datadogQuery.Search != nil {
		terraformQuery["search"] = map[string]interface{}{
			"query": *datadogQuery.Search.Query,
		}
	}
	// GroupBy
	if datadogQuery.GroupBy != nil {
		terraformGroupBys := make([]map[string]interface{}, len(datadogQuery.GroupBy))
		for i, groupBy := range datadogQuery.GroupBy {
			// Facet
			terraformGroupBy := map[string]interface{}{
				"facet": *groupBy.Facet,
			}
			// Limit
			if groupBy.Limit != nil {
				terraformGroupBy["limit"] = *groupBy.Limit
			}
			// Sort
			if groupBy.Sort != nil {
				sort := map[string]string{
					"aggregation": *groupBy.Sort.Aggregation,
					"order":       *groupBy.Sort.Order,
				}
				if groupBy.Sort.Facet != nil {
					sort["facet"] = *groupBy.Sort.Facet
				}
				terraformGroupBy["sort"] = sort
			}
			terraformGroupBys[i] = terraformGroupBy
		}
		terraformQuery["group_by"] = &terraformGroupBys
	}
	return terraformQuery
}

// Process Query
func getProcessQuerySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"metric": {
					Type:     schema.TypeString,
					Required: true,
				},
				"search_by": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"filter_by": {
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
				"limit": {
					Type:     schema.TypeInt,
					Optional: true,
				},
			},
		},
	}
}
func buildDatadogProcessQuery(terraformQuery map[string]interface{}) *datadog.WidgetProcessQuery {
	datadogQuery := datadog.WidgetProcessQuery{}
	if v, ok := terraformQuery["metric"].(string); ok && len(v) != 0 {
		datadogQuery.SetMetric(v)
	}
	if v, ok := terraformQuery["search_by"].(string); ok && len(v) != 0 {
		datadogQuery.SetSearchBy(v)
	}

	if terraformFilterBys, ok := terraformQuery["filter_by"].([]interface{}); ok && len(terraformFilterBys) > 0 {
		datadogFilterbys := make([]string, len(terraformFilterBys))
		for i, filtrBy := range terraformFilterBys {
			datadogFilterbys[i] = filtrBy.(string)
		}
		datadogQuery.FilterBy = datadogFilterbys
	}

	if v, ok := terraformQuery["limit"].(int); ok && v != 0 {
		datadogQuery.SetLimit(v)
	}

	return &datadogQuery
}

func buildTerraformProcessQuery(datadogQuery datadog.WidgetProcessQuery) map[string]interface{} {
	terraformQuery := map[string]interface{}{}
	if datadogQuery.Metric != nil {
		terraformQuery["metric"] = *datadogQuery.Metric
	}
	if datadogQuery.SearchBy != nil {
		terraformQuery["search_by"] = *datadogQuery.SearchBy
	}
	if datadogQuery.FilterBy != nil {
		terraformFilterBys := make([]string, len(datadogQuery.FilterBy))
		for i, datadogFilterBy := range datadogQuery.FilterBy {
			terraformFilterBys[i] = datadogFilterBy
		}
		terraformQuery["filter_by"] = terraformFilterBys
	}
	if datadogQuery.Limit != nil {
		terraformQuery["limit"] = *datadogQuery.Limit
	}

	return terraformQuery
}
