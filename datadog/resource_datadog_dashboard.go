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
	if _def, ok := terraformWidget["note_definition"].([]interface{}); ok && len(_def) > 0 {
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
	// case datadog.ALERT_GRAPH_WIDGET:
	// 	datadogDefinition := datadogWidget.Definition.(datadog.AlertGraphDefinition)
	// 	terraformDefinition := buildTerraformAlertGraphDefinition(datadogDefinition)
	// 	terraformWidget["alert_graph_definition"] = []map[string]interface{}{terraformDefinition}
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
