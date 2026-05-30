package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/sla"
)

// ----------------------------------------------------------------------------
// Local output views
// ----------------------------------------------------------------------------
//
// sla.SLATemplateResponse cannot be reused as an MCP Output: its PriorityLevels,
// SeverityLevels, Holidays ([]string) and ResponseTimes, ResolutionTimes,
// BusinessHours, Configuration (map) fields lack `omitempty`, so a nil value
// marshals to JSON null and the go-sdk rejects it against the inferred
// array/object output schema (breaking the success path on nil data and every
// error path, which returns a zero Out). slaTemplateResponse is the schema-safe
// MCP view: every slice/map field carries `omitempty`.
//
// sla.SLARuleResponse has no slice/map fields, but slaRuleResponse mirrors it as
// an MCP-local view so the whole package uniformly returns MCP-local Output
// structs; list slices carry `omitempty` so a nil page is omitted, not null.
type slaTemplateResponse struct {
	ID              uint                   `json:"id" jsonschema:"the template's numeric ID"`
	Name            string                 `json:"name" jsonschema:"the SLA template name"`
	Description     string                 `json:"description,omitempty" jsonschema:"the template description"`
	IsDefault       bool                   `json:"is_default" jsonschema:"whether this template is the default"`
	IsActive        bool                   `json:"is_active" jsonschema:"whether the template is active"`
	PriorityLevels  []string               `json:"priority_levels,omitempty" jsonschema:"the priority levels"`
	SeverityLevels  []string               `json:"severity_levels,omitempty" jsonschema:"the severity levels"`
	ResponseTimes   map[string]interface{} `json:"response_times,omitempty" jsonschema:"response time targets keyed by level"`
	ResolutionTimes map[string]interface{} `json:"resolution_times,omitempty" jsonschema:"resolution time targets keyed by level"`
	BusinessHours   map[string]interface{} `json:"business_hours,omitempty" jsonschema:"business hours configuration"`
	Holidays        []string               `json:"holidays,omitempty" jsonschema:"holiday dates"`
	Configuration   map[string]interface{} `json:"configuration,omitempty" jsonschema:"additional configuration"`
	IsDeleted       bool                   `json:"is_deleted" jsonschema:"whether the template is soft-deleted"`
	CreatedAt       time.Time              `json:"created_at" jsonschema:"when the template was created"`
	UpdatedAt       time.Time              `json:"updated_at" jsonschema:"when the template was last updated"`
}

// slaTemplateResponseFrom converts a service-layer sla.SLATemplateResponse into
// the schema-safe MCP view.
func slaTemplateResponseFrom(r *sla.SLATemplateResponse) slaTemplateResponse {
	if r == nil {
		return slaTemplateResponse{}
	}
	return slaTemplateResponse{
		ID:              r.ID,
		Name:            r.Name,
		Description:     r.Description,
		IsDefault:       r.IsDefault,
		IsActive:        r.IsActive,
		PriorityLevels:  r.PriorityLevels,
		SeverityLevels:  r.SeverityLevels,
		ResponseTimes:   r.ResponseTimes,
		ResolutionTimes: r.ResolutionTimes,
		BusinessHours:   r.BusinessHours,
		Holidays:        r.Holidays,
		Configuration:   r.Configuration,
		IsDeleted:       r.IsDeleted,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

// slaTemplateResponsesFrom converts a slice of service-layer responses into views.
func slaTemplateResponsesFrom(rs []sla.SLATemplateResponse) []slaTemplateResponse {
	if len(rs) == 0 {
		return nil
	}
	views := make([]slaTemplateResponse, len(rs))
	for i := range rs {
		views[i] = slaTemplateResponseFrom(&rs[i])
	}
	return views
}

// slaRuleResponse is the MCP-local view of an SLA rule. sla.SLARuleResponse has
// no slice/map fields, so this is a 1:1 flat copy; it exists so the package
// follows the uniform rule that every tool Output is an MCP-local struct rather
// than a service-layer DTO. Its pointer fields carry omitempty for tidiness.
type slaRuleResponse struct {
	ID             uint      `json:"id" jsonschema:"the rule's numeric ID"`
	TemplateID     uint      `json:"template_id" jsonschema:"the owning SLA template ID"`
	Priority       string    `json:"priority" jsonschema:"the ticket priority this rule matches"`
	Severity       string    `json:"severity" jsonschema:"the ticket severity this rule matches"`
	ResponseTime   int       `json:"response_time" jsonschema:"target response time in minutes"`
	ResolutionTime int       `json:"resolution_time" jsonschema:"target resolution time in minutes"`
	BusinessOnly   bool      `json:"business_only" jsonschema:"whether timing counts only business hours"`
	ProductID      *uint     `json:"product_id,omitempty" jsonschema:"optional product scope"`
	ServiceID      *uint     `json:"service_id,omitempty" jsonschema:"optional service scope"`
	Conditions     string    `json:"conditions,omitempty" jsonschema:"additional matching conditions as a JSON string"`
	IsActive       bool      `json:"is_active" jsonschema:"whether the rule is active"`
	IsDeleted      bool      `json:"is_deleted" jsonschema:"whether the rule is soft-deleted"`
	CreatedAt      time.Time `json:"created_at" jsonschema:"when the rule was created"`
	UpdatedAt      time.Time `json:"updated_at" jsonschema:"when the rule was last updated"`
}

// slaRuleResponseFrom converts a service-layer sla.SLARuleResponse into the view.
func slaRuleResponseFrom(r *sla.SLARuleResponse) slaRuleResponse {
	if r == nil {
		return slaRuleResponse{}
	}
	return slaRuleResponse{
		ID:             r.ID,
		TemplateID:     r.TemplateID,
		Priority:       r.Priority,
		Severity:       r.Severity,
		ResponseTime:   r.ResponseTime,
		ResolutionTime: r.ResolutionTime,
		BusinessOnly:   r.BusinessOnly,
		ProductID:      r.ProductID,
		ServiceID:      r.ServiceID,
		Conditions:     r.Conditions,
		IsActive:       r.IsActive,
		IsDeleted:      r.IsDeleted,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

// slaRuleResponsesFrom converts a slice of service-layer responses into views.
func slaRuleResponsesFrom(rs []sla.SLARuleResponse) []slaRuleResponse {
	if len(rs) == 0 {
		return nil
	}
	views := make([]slaRuleResponse, len(rs))
	for i := range rs {
		views[i] = slaRuleResponseFrom(&rs[i])
	}
	return views
}

// registerSLATools registers the SLA-domain MCP tools: 8 template tools and 7
// rule tools. See server.go for the tool implementation conventions and the
// auth_whoami template. Read operations (get/list) require "sla:read"; write
// operations (create/update/delete/set_default/activate/deactivate) require
// "sla:write".
func registerSLATools(s *mcp.Server, b Backend) {
	// --- SLA template tools ---

	registerTool(s,
		"sla_create_template",
		"Create a new SLA template defining priority/severity levels and response/resolution targets.",
		"sla:write",
		func(ctx context.Context, in slaCreateTemplateInput) (slaTemplateResponse, string, error) {
			return slaCreateTemplate(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_get_template",
		"Retrieve a single SLA template by its numeric ID.",
		"sla:read",
		func(ctx context.Context, in slaGetTemplateInput) (slaTemplateResponse, string, error) {
			return slaGetTemplate(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_list_templates",
		"List SLA templates with pagination, search, and active-state filtering.",
		"sla:read",
		func(ctx context.Context, in slaListTemplatesInput) (slaTemplateListOutput, string, error) {
			return slaListTemplates(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_update_template",
		"Update fields of an existing SLA template. Only provided fields are changed.",
		"sla:write",
		func(ctx context.Context, in slaUpdateTemplateInput) (slaTemplateResponse, string, error) {
			return slaUpdateTemplate(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_delete_template",
		"Soft delete an SLA template by ID.",
		"sla:write",
		func(ctx context.Context, in slaTemplateIDInput) (slaActionOutput, string, error) {
			return slaDeleteTemplate(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_set_default_template",
		"Mark an SLA template as the default, unsetting any previous default.",
		"sla:write",
		func(ctx context.Context, in slaTemplateIDInput) (slaActionOutput, string, error) {
			return slaSetDefaultTemplate(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_activate_template",
		"Activate an SLA template by ID.",
		"sla:write",
		func(ctx context.Context, in slaTemplateIDInput) (slaActionOutput, string, error) {
			return slaActivateTemplate(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_deactivate_template",
		"Deactivate an SLA template by ID.",
		"sla:write",
		func(ctx context.Context, in slaTemplateIDInput) (slaActionOutput, string, error) {
			return slaDeactivateTemplate(ctx, b, in)
		},
	)

	// --- SLA rule tools ---

	registerTool(s,
		"sla_create_rule",
		"Create a new SLA rule binding a priority/severity (and optional product/service) to response/resolution times.",
		"sla:write",
		func(ctx context.Context, in slaCreateRuleInput) (slaRuleResponse, string, error) {
			return slaCreateRule(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_get_rule",
		"Retrieve a single SLA rule by its numeric ID.",
		"sla:read",
		func(ctx context.Context, in slaGetRuleInput) (slaRuleResponse, string, error) {
			return slaGetRule(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_list_rules",
		"List SLA rules with pagination and filtering by priority, severity, product, service, and active state.",
		"sla:read",
		func(ctx context.Context, in slaListRulesInput) (slaRuleListOutput, string, error) {
			return slaListRules(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_update_rule",
		"Update fields of an existing SLA rule. Only provided fields are changed.",
		"sla:write",
		func(ctx context.Context, in slaUpdateRuleInput) (slaRuleResponse, string, error) {
			return slaUpdateRule(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_delete_rule",
		"Soft delete an SLA rule by ID.",
		"sla:write",
		func(ctx context.Context, in slaRuleIDInput) (slaActionOutput, string, error) {
			return slaDeleteRule(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_activate_rule",
		"Activate an SLA rule by ID.",
		"sla:write",
		func(ctx context.Context, in slaRuleIDInput) (slaActionOutput, string, error) {
			return slaActivateRule(ctx, b, in)
		},
	)

	registerTool(s,
		"sla_deactivate_rule",
		"Deactivate an SLA rule by ID.",
		"sla:write",
		func(ctx context.Context, in slaRuleIDInput) (slaActionOutput, string, error) {
			return slaDeactivateRule(ctx, b, in)
		},
	)
}

// ----------------------------------------------------------------------------
// Shared SLA output types
// ----------------------------------------------------------------------------

// slaActionOutput is the structured output for SLA tools that perform an action
// without returning an entity (delete/activate/deactivate/set_default).
type slaActionOutput struct {
	ID      uint   `json:"id" jsonschema:"the numeric ID of the affected SLA entity"`
	Status  string `json:"status" jsonschema:"the result status, e.g. deleted or activated"`
	Message string `json:"message" jsonschema:"a human-readable summary of the action"`
}

// slaTemplateListOutput is the structured output for sla_list_templates.
type slaTemplateListOutput struct {
	Templates []slaTemplateResponse `json:"templates,omitempty" jsonschema:"the page of SLA templates"`
	Total     int64                 `json:"total" jsonschema:"total number of matching SLA templates"`
	Page      int                   `json:"page" jsonschema:"the current page number"`
	PageSize  int                   `json:"page_size" jsonschema:"the page size used"`
}

// slaRuleListOutput is the structured output for sla_list_rules.
type slaRuleListOutput struct {
	Rules    []slaRuleResponse `json:"rules,omitempty" jsonschema:"the page of SLA rules"`
	Total    int64             `json:"total" jsonschema:"total number of matching SLA rules"`
	Page     int               `json:"page" jsonschema:"the current page number"`
	PageSize int               `json:"page_size" jsonschema:"the page size used"`
}

// ----------------------------------------------------------------------------
// SLA template tools
// ----------------------------------------------------------------------------

// slaCreateTemplateInput is the input schema for sla_create_template. The JSON
// fields (priority_levels, response_times, etc.) carry serialized JSON payloads
// as expected by the service layer.
type slaCreateTemplateInput struct {
	Name            string `json:"name" jsonschema:"the unique SLA template name (required)"`
	Description     string `json:"description,omitempty" jsonschema:"a human-readable description"`
	IsDefault       bool   `json:"is_default,omitempty" jsonschema:"whether this template is the default"`
	IsActive        bool   `json:"is_active,omitempty" jsonschema:"whether the template is active"`
	PriorityLevels  string `json:"priority_levels,omitempty" jsonschema:"JSON array of priority levels"`
	SeverityLevels  string `json:"severity_levels,omitempty" jsonschema:"JSON array of severity levels"`
	ResponseTimes   string `json:"response_times,omitempty" jsonschema:"JSON object mapping levels to response time targets"`
	ResolutionTimes string `json:"resolution_times,omitempty" jsonschema:"JSON object mapping levels to resolution time targets"`
	BusinessHours   string `json:"business_hours,omitempty" jsonschema:"JSON object describing business hours"`
	Holidays        string `json:"holidays,omitempty" jsonschema:"JSON array of holiday dates"`
	Configuration   string `json:"configuration,omitempty" jsonschema:"JSON object of additional configuration"`
}

func slaCreateTemplate(_ context.Context, b Backend, in slaCreateTemplateInput) (slaTemplateResponse, string, error) {
	req := &sla.CreateSLATemplateRequest{
		Name:            in.Name,
		Description:     in.Description,
		IsDefault:       in.IsDefault,
		IsActive:        in.IsActive,
		PriorityLevels:  in.PriorityLevels,
		SeverityLevels:  in.SeverityLevels,
		ResponseTimes:   in.ResponseTimes,
		ResolutionTimes: in.ResolutionTimes,
		BusinessHours:   in.BusinessHours,
		Holidays:        in.Holidays,
		Configuration:   in.Configuration,
	}
	resp, err := b.CreateSLATemplate(req)
	if err != nil {
		return slaTemplateResponse{}, "", err
	}
	return slaTemplateResponseFrom(resp), fmt.Sprintf("Created SLA template #%d (%s).", resp.ID, resp.Name), nil
}

// slaGetTemplateInput is the input schema for sla_get_template.
type slaGetTemplateInput struct {
	TemplateID uint `json:"template_id" jsonschema:"the numeric ID of the SLA template to retrieve"`
}

func slaGetTemplate(_ context.Context, b Backend, in slaGetTemplateInput) (slaTemplateResponse, string, error) {
	resp, err := b.GetSLATemplate(in.TemplateID)
	if err != nil {
		return slaTemplateResponse{}, "", err
	}
	return slaTemplateResponseFrom(resp), fmt.Sprintf("Retrieved SLA template #%d (%s).", resp.ID, resp.Name), nil
}

// slaListTemplatesInput is the input schema for sla_list_templates.
type slaListTemplatesInput struct {
	Page      int    `json:"page,omitempty" jsonschema:"the 1-based page number (default 1)"`
	PageSize  int    `json:"page_size,omitempty" jsonschema:"the page size, 1-100 (default 20)"`
	Search    string `json:"search,omitempty" jsonschema:"case-insensitive search over name and description"`
	IsActive  *bool  `json:"is_active,omitempty" jsonschema:"filter by active state"`
	SortBy    string `json:"sort_by,omitempty" jsonschema:"field to sort by (default created_at)"`
	SortOrder string `json:"sort_order,omitempty" jsonschema:"sort direction asc or desc (default desc)"`
}

func slaListTemplates(_ context.Context, b Backend, in slaListTemplatesInput) (slaTemplateListOutput, string, error) {
	req := &sla.ListSLATemplatesRequest{
		Page:      in.Page,
		PageSize:  in.PageSize,
		Search:    in.Search,
		IsActive:  in.IsActive,
		SortBy:    in.SortBy,
		SortOrder: in.SortOrder,
	}
	list, total, err := b.ListSLATemplates(req)
	if err != nil {
		return slaTemplateListOutput{}, "", err
	}
	out := slaTemplateListOutput{
		Templates: slaTemplateResponsesFrom(list),
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}
	return out, fmt.Sprintf("Listed %d of %d SLA template(s).", len(list), total), nil
}

// slaUpdateTemplateInput is the input schema for sla_update_template. All
// fields are optional pointers so only provided values are changed.
type slaUpdateTemplateInput struct {
	TemplateID      uint    `json:"template_id" jsonschema:"the numeric ID of the SLA template to update"`
	Name            *string `json:"name,omitempty" jsonschema:"new template name"`
	Description     *string `json:"description,omitempty" jsonschema:"new description"`
	IsDefault       *bool   `json:"is_default,omitempty" jsonschema:"whether this template is the default"`
	IsActive        *bool   `json:"is_active,omitempty" jsonschema:"whether the template is active"`
	PriorityLevels  *string `json:"priority_levels,omitempty" jsonschema:"JSON array of priority levels"`
	SeverityLevels  *string `json:"severity_levels,omitempty" jsonschema:"JSON array of severity levels"`
	ResponseTimes   *string `json:"response_times,omitempty" jsonschema:"JSON object of response time targets"`
	ResolutionTimes *string `json:"resolution_times,omitempty" jsonschema:"JSON object of resolution time targets"`
	BusinessHours   *string `json:"business_hours,omitempty" jsonschema:"JSON object describing business hours"`
	Holidays        *string `json:"holidays,omitempty" jsonschema:"JSON array of holiday dates"`
	Configuration   *string `json:"configuration,omitempty" jsonschema:"JSON object of additional configuration"`
}

func slaUpdateTemplate(_ context.Context, b Backend, in slaUpdateTemplateInput) (slaTemplateResponse, string, error) {
	req := &sla.UpdateSLATemplateRequest{
		Name:            in.Name,
		Description:     in.Description,
		IsDefault:       in.IsDefault,
		IsActive:        in.IsActive,
		PriorityLevels:  in.PriorityLevels,
		SeverityLevels:  in.SeverityLevels,
		ResponseTimes:   in.ResponseTimes,
		ResolutionTimes: in.ResolutionTimes,
		BusinessHours:   in.BusinessHours,
		Holidays:        in.Holidays,
		Configuration:   in.Configuration,
	}
	resp, err := b.UpdateSLATemplate(in.TemplateID, req)
	if err != nil {
		return slaTemplateResponse{}, "", err
	}
	return slaTemplateResponseFrom(resp), fmt.Sprintf("Updated SLA template #%d (%s).", resp.ID, resp.Name), nil
}

// slaTemplateIDInput is the input schema for template tools that act on an ID.
type slaTemplateIDInput struct {
	TemplateID uint `json:"template_id" jsonschema:"the numeric ID of the SLA template"`
}

func slaDeleteTemplate(_ context.Context, b Backend, in slaTemplateIDInput) (slaActionOutput, string, error) {
	if err := b.DeleteSLATemplate(in.TemplateID); err != nil {
		return slaActionOutput{}, "", err
	}
	summary := fmt.Sprintf("Deleted SLA template #%d.", in.TemplateID)
	return slaActionOutput{ID: in.TemplateID, Status: "deleted", Message: summary}, summary, nil
}

func slaSetDefaultTemplate(_ context.Context, b Backend, in slaTemplateIDInput) (slaActionOutput, string, error) {
	if err := b.SetDefaultSLATemplate(in.TemplateID); err != nil {
		return slaActionOutput{}, "", err
	}
	summary := fmt.Sprintf("Set SLA template #%d as default.", in.TemplateID)
	return slaActionOutput{ID: in.TemplateID, Status: "default", Message: summary}, summary, nil
}

func slaActivateTemplate(_ context.Context, b Backend, in slaTemplateIDInput) (slaActionOutput, string, error) {
	if err := b.ActivateSLATemplate(in.TemplateID); err != nil {
		return slaActionOutput{}, "", err
	}
	summary := fmt.Sprintf("Activated SLA template #%d.", in.TemplateID)
	return slaActionOutput{ID: in.TemplateID, Status: "activated", Message: summary}, summary, nil
}

func slaDeactivateTemplate(_ context.Context, b Backend, in slaTemplateIDInput) (slaActionOutput, string, error) {
	if err := b.DeactivateSLATemplate(in.TemplateID); err != nil {
		return slaActionOutput{}, "", err
	}
	summary := fmt.Sprintf("Deactivated SLA template #%d.", in.TemplateID)
	return slaActionOutput{ID: in.TemplateID, Status: "deactivated", Message: summary}, summary, nil
}

// ----------------------------------------------------------------------------
// SLA rule tools
// ----------------------------------------------------------------------------

// slaCreateRuleInput is the input schema for sla_create_rule.
type slaCreateRuleInput struct {
	TemplateID     uint   `json:"template_id" jsonschema:"the SLA template this rule belongs to (required)"`
	Priority       string `json:"priority" jsonschema:"the ticket priority: low, medium, high, or critical (required)"`
	Severity       string `json:"severity" jsonschema:"the ticket severity: trivial, minor, major, or critical (required)"`
	ResponseTime   int    `json:"response_time" jsonschema:"target response time in minutes (required, min 1)"`
	ResolutionTime int    `json:"resolution_time" jsonschema:"target resolution time in minutes (required, min 1)"`
	BusinessOnly   bool   `json:"business_only,omitempty" jsonschema:"whether timing counts only business hours"`
	ProductID      *uint  `json:"product_id,omitempty" jsonschema:"optional product scope for this rule"`
	ServiceID      *uint  `json:"service_id,omitempty" jsonschema:"optional service scope for this rule"`
	Conditions     string `json:"conditions,omitempty" jsonschema:"JSON object of additional matching conditions"`
}

func slaCreateRule(_ context.Context, b Backend, in slaCreateRuleInput) (slaRuleResponse, string, error) {
	req := &sla.CreateSLARuleRequest{
		TemplateID:     in.TemplateID,
		Priority:       in.Priority,
		Severity:       in.Severity,
		ResponseTime:   in.ResponseTime,
		ResolutionTime: in.ResolutionTime,
		BusinessOnly:   in.BusinessOnly,
		ProductID:      in.ProductID,
		ServiceID:      in.ServiceID,
		Conditions:     in.Conditions,
	}
	resp, err := b.CreateSLARule(req)
	if err != nil {
		return slaRuleResponse{}, "", err
	}
	return slaRuleResponseFrom(resp), fmt.Sprintf("Created SLA rule #%d (%s/%s).", resp.ID, resp.Priority, resp.Severity), nil
}

// slaGetRuleInput is the input schema for sla_get_rule.
type slaGetRuleInput struct {
	RuleID uint `json:"rule_id" jsonschema:"the numeric ID of the SLA rule to retrieve"`
}

func slaGetRule(_ context.Context, b Backend, in slaGetRuleInput) (slaRuleResponse, string, error) {
	resp, err := b.GetSLARule(in.RuleID)
	if err != nil {
		return slaRuleResponse{}, "", err
	}
	return slaRuleResponseFrom(resp), fmt.Sprintf("Retrieved SLA rule #%d (%s/%s).", resp.ID, resp.Priority, resp.Severity), nil
}

// slaListRulesInput is the input schema for sla_list_rules.
type slaListRulesInput struct {
	Page      int    `json:"page,omitempty" jsonschema:"the 1-based page number (default 1)"`
	PageSize  int    `json:"page_size,omitempty" jsonschema:"the page size, 1-100 (default 20)"`
	Search    string `json:"search,omitempty" jsonschema:"search over rule conditions"`
	IsActive  *bool  `json:"is_active,omitempty" jsonschema:"filter by active state"`
	Priority  string `json:"priority,omitempty" jsonschema:"filter by priority"`
	Severity  string `json:"severity,omitempty" jsonschema:"filter by severity"`
	ProductID *uint  `json:"product_id,omitempty" jsonschema:"filter by product scope"`
	ServiceID *uint  `json:"service_id,omitempty" jsonschema:"filter by service scope"`
	SortBy    string `json:"sort_by,omitempty" jsonschema:"field to sort by (default created_at)"`
	SortOrder string `json:"sort_order,omitempty" jsonschema:"sort direction asc or desc (default desc)"`
}

func slaListRules(_ context.Context, b Backend, in slaListRulesInput) (slaRuleListOutput, string, error) {
	req := &sla.ListSLARulesRequest{
		Page:      in.Page,
		PageSize:  in.PageSize,
		Search:    in.Search,
		IsActive:  in.IsActive,
		Priority:  in.Priority,
		Severity:  in.Severity,
		ProductID: in.ProductID,
		ServiceID: in.ServiceID,
		SortBy:    in.SortBy,
		SortOrder: in.SortOrder,
	}
	list, total, err := b.ListSLARules(req)
	if err != nil {
		return slaRuleListOutput{}, "", err
	}
	out := slaRuleListOutput{
		Rules:    slaRuleResponsesFrom(list),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	return out, fmt.Sprintf("Listed %d of %d SLA rule(s).", len(list), total), nil
}

// slaUpdateRuleInput is the input schema for sla_update_rule. All mutable
// fields are optional pointers so only provided values are changed.
type slaUpdateRuleInput struct {
	RuleID         uint    `json:"rule_id" jsonschema:"the numeric ID of the SLA rule to update"`
	TemplateID     *uint   `json:"template_id,omitempty" jsonschema:"new SLA template ID"`
	Priority       *string `json:"priority,omitempty" jsonschema:"new priority: low, medium, high, or critical"`
	Severity       *string `json:"severity,omitempty" jsonschema:"new severity: trivial, minor, major, or critical"`
	ResponseTime   *int    `json:"response_time,omitempty" jsonschema:"new response time in minutes (min 1)"`
	ResolutionTime *int    `json:"resolution_time,omitempty" jsonschema:"new resolution time in minutes (min 1)"`
	BusinessOnly   *bool   `json:"business_only,omitempty" jsonschema:"whether timing counts only business hours"`
	ProductID      *uint   `json:"product_id,omitempty" jsonschema:"new product scope"`
	ServiceID      *uint   `json:"service_id,omitempty" jsonschema:"new service scope"`
	Conditions     *string `json:"conditions,omitempty" jsonschema:"JSON object of additional matching conditions"`
}

func slaUpdateRule(_ context.Context, b Backend, in slaUpdateRuleInput) (slaRuleResponse, string, error) {
	req := &sla.UpdateSLARuleRequest{
		TemplateID:     in.TemplateID,
		Priority:       in.Priority,
		Severity:       in.Severity,
		ResponseTime:   in.ResponseTime,
		ResolutionTime: in.ResolutionTime,
		BusinessOnly:   in.BusinessOnly,
		ProductID:      in.ProductID,
		ServiceID:      in.ServiceID,
		Conditions:     in.Conditions,
	}
	resp, err := b.UpdateSLARule(in.RuleID, req)
	if err != nil {
		return slaRuleResponse{}, "", err
	}
	return slaRuleResponseFrom(resp), fmt.Sprintf("Updated SLA rule #%d (%s/%s).", resp.ID, resp.Priority, resp.Severity), nil
}

// slaRuleIDInput is the input schema for rule tools that act on an ID.
type slaRuleIDInput struct {
	RuleID uint `json:"rule_id" jsonschema:"the numeric ID of the SLA rule"`
}

func slaDeleteRule(_ context.Context, b Backend, in slaRuleIDInput) (slaActionOutput, string, error) {
	if err := b.DeleteSLARule(in.RuleID); err != nil {
		return slaActionOutput{}, "", err
	}
	summary := fmt.Sprintf("Deleted SLA rule #%d.", in.RuleID)
	return slaActionOutput{ID: in.RuleID, Status: "deleted", Message: summary}, summary, nil
}

func slaActivateRule(_ context.Context, b Backend, in slaRuleIDInput) (slaActionOutput, string, error) {
	if err := b.ActivateSLARule(in.RuleID); err != nil {
		return slaActionOutput{}, "", err
	}
	summary := fmt.Sprintf("Activated SLA rule #%d.", in.RuleID)
	return slaActionOutput{ID: in.RuleID, Status: "activated", Message: summary}, summary, nil
}

func slaDeactivateRule(_ context.Context, b Backend, in slaRuleIDInput) (slaActionOutput, string, error) {
	if err := b.DeactivateSLARule(in.RuleID); err != nil {
		return slaActionOutput{}, "", err
	}
	summary := fmt.Sprintf("Deactivated SLA rule #%d.", in.RuleID)
	return slaActionOutput{ID: in.RuleID, Status: "deactivated", Message: summary}, summary, nil
}
