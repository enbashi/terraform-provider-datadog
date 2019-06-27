package datadog

import (
	"fmt"
	"strconv"

	datadog "github.com/MLaureB/go-datadog-api"
	"github.com/hashicorp/terraform/helper/schema"
)

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
	for i, _templateVariable := range *terraformTemplateVariables {
		templateVariable := _templateVariable.(map[string]interface{})
		datadogTemplateVariables[i] = datadog.TemplateVariable{
			Name:    datadog.String(templateVariable["name"].(string)),
			Prefix:  datadog.String(templateVariable["prefix"].(string)),
			Default: datadog.String(templateVariable["default"].(string)),
		}
	}
	return &datadogTemplateVariables
}

func buildTerraformTemplateVariables(datadogTemplateVariables *[]datadog.TemplateVariable) *[]map[string]string {
	terraformTemplateVariables := make([]map[string]string, len(*datadogTemplateVariables))
	for i, templateVariable := range *datadogTemplateVariables {
		terraformTemplateVariable := map[string]string{}
		// Required params
		terraformTemplateVariable["name"] = *templateVariable.Name
		// Optional params
		if templateVariable.Prefix != nil {
			terraformTemplateVariable["prefix"] = *templateVariable.Prefix
		}
		if templateVariable.Default != nil {
			terraformTemplateVariable["default"] = *templateVariable.Default
		}
		terraformTemplateVariables[i] = terraformTemplateVariable
	}
	return &terraformTemplateVariables
}

//
// Notify List helpers
//

func buildDatadogNotifyList(terraformNotifyList *[]interface{}) []string {
	datadogNotifyList := make([]string, len(*terraformNotifyList))
	for i, authorHandle := range *terraformNotifyList {
		datadogNotifyList[i] = authorHandle.(string)
	}
	return datadogNotifyList
}

func buildTerraformNotifyList(datadogNotifyList *[]string) []string {
	terraformNotifyList := make([]string, len(*datadogNotifyList))
	for i, authorHandle := range *datadogNotifyList {
		terraformNotifyList[i] = authorHandle
	}
	return terraformNotifyList
}

//
// Widgets helpers
//

// The generic widget schema is a combinaison of the schema for a non-group widget
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

// Schema for a non-group widget
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
		"alert_graph_definition": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The definition for an Alert Graph widget",
			Elem: &schema.Resource{
				Schema: getAlertGraphDefinitionSchema(),
			},
		},
		"note_definition": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The definition for a Note widget",
			Elem: &schema.Resource{
				Schema: getNoteDefinitionSchema(),
			},
		},
	}
}

// Helper to build a list of Datadog widgets from a list of Terraform widgets
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

	// Build widget Layout
	if layout, ok := terraformWidget["layout"].(map[string]interface{}); ok && len(layout) > 0 {
		datadogWidget.Layout = buildDatadogWidgetLayout(layout)
	}

	// Build widget Definition
	if _def, ok := terraformWidget["alert_graph_definition"].([]interface{}); ok && len(_def) > 0 {
		if alertGraphDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogWidget.Definition = buildDatadogAlertGraphDefinition(alertGraphDefinition)
		}
	} else if _def, ok := terraformWidget["note_definition"].([]interface{}); ok && len(_def) > 0 {
		if noteDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogWidget.Definition = buildDatadogNoteDefinition(noteDefinition)
		}
	} else if _def, ok := terraformWidget["group_definition"].([]interface{}); ok && len(_def) > 0 {
		if groupDefinition, ok := _def[0].(map[string]interface{}); ok {
			datadogDefinition, err := buildDatadogGroupDefinition(groupDefinition)
			if err != nil {
				return nil, err
			}
			datadogWidget.Definition = datadogDefinition
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
	case datadog.ALERT_GRAPH_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.AlertGraphDefinition)
		terraformDefinition := buildTerraformAlertGraphDefinition(datadogDefinition)
		terraformWidget["alert_graph_definition"] = []map[string]interface{}{terraformDefinition}
	case datadog.NOTE_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.NoteDefinition)
		terraformDefinition := buildTerraformNoteDefinition(datadogDefinition)
		terraformWidget["note_definition"] = []map[string]interface{}{terraformDefinition}
	case datadog.GROUP_WIDGET:
		datadogDefinition := datadogWidget.Definition.(datadog.GroupDefinition)
		terraformDefinition := buildTerraformGroupDefinition(datadogDefinition)
		terraformWidget["group_definition"] = []map[string]interface{}{terraformDefinition}
	default:
		return nil, fmt.Errorf("Unsupported widget type: %s", widgetType)
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

func buildDatadogWidgetLayout(terraformLayout map[string]interface{}) *datadog.WidgetLayout {
	datadogLayout := &datadog.WidgetLayout{}
	if v, err := strconv.ParseFloat(terraformLayout["x"].(string), 64); err == nil {
		datadogLayout.X = &v
	}
	if v, err := strconv.ParseFloat(terraformLayout["y"].(string), 64); err == nil {
		datadogLayout.Y = &v
	}
	if v, err := strconv.ParseFloat(terraformLayout["height"].(string), 64); err == nil {
		datadogLayout.Height = &v
	}
	if v, err := strconv.ParseFloat(terraformLayout["width"].(string), 64); err == nil {
		datadogLayout.Width = &v
	}
	return datadogLayout
}

func buildTerraformWidgetLayout(datadogLayout datadog.WidgetLayout) map[string]string {
	terraformLayout := map[string]string{}
	if datadogLayout.X != nil {
		terraformLayout["x"] = strconv.FormatFloat(*datadogLayout.X, 'f', -1, 64)
	}
	if datadogLayout.Y != nil {
		terraformLayout["y"] = strconv.FormatFloat(*datadogLayout.Y, 'f', -1, 64)
	}
	if datadogLayout.Height != nil {
		terraformLayout["height"] = strconv.FormatFloat(*datadogLayout.Height, 'f', -1, 64)
	}
	if datadogLayout.Width != nil {
		terraformLayout["width"] = strconv.FormatFloat(*datadogLayout.Width, 'f', -1, 64)
	}
	return terraformLayout
}

//
// Alert Graph Definition helpers
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
// Group Widget Definition helpers
//

func getGroupDefinitionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"layout_type": {
			Type:     schema.TypeString,
			Required: true,
		},
		"widget": {
			Type:        schema.TypeList,
			Required:    true,
			Description: "The list of widgets in this group.",
			Elem: &schema.Resource{
				Schema: getNonGroupWidgetSchema(),
			},
		},
		"title": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}
}

func buildDatadogGroupDefinition(terraformDefinition map[string]interface{}) (*datadog.GroupDefinition, error) {
	datadogDefinition := &datadog.GroupDefinition{}
	// Required params
	datadogDefinition.Type = datadog.String(datadog.GROUP_WIDGET)
	if v, ok := terraformDefinition["layout_type"].(string); ok && len(v) != 0 {
		datadogDefinition.LayoutType = datadog.String(v)
	}
	if v, ok := terraformDefinition["widget"].([]interface{}); ok {
		groupWidgets, err := buildDatadogWidgets(&v)
		if err != nil {
			return nil, err
		}
		datadogDefinition.Widgets = *groupWidgets
	}
	// Optional params
	if v, ok := terraformDefinition["title"].(string); ok && len(v) != 0 {
		datadogDefinition.Title = datadog.String(v)
	}
	return datadogDefinition, nil
}

func buildTerraformGroupDefinition(datadogDefinition datadog.GroupDefinition) map[string]interface{} {
	terraformDefinition := map[string]interface{}{}
	// Required params
	terraformDefinition["layout_type"] = *datadogDefinition.LayoutType
	groupWidgets := []map[string]interface{}{}
	for _, datadogGroupWidgets := range datadogDefinition.Widgets {
		newGroupWidget, _ := buildTerraformWidget(datadogGroupWidgets)
		groupWidgets = append(groupWidgets, newGroupWidget)
	}
	terraformDefinition["widget"] = groupWidgets
	// Optional params
	if datadogDefinition.Title != nil {
		terraformDefinition["title"] = *datadogDefinition.Title
	}
	return terraformDefinition
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
// Helpers common to different widget definitions
//

// Widget Time
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
