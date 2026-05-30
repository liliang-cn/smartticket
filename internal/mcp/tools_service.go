package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	servicemgmt "github.com/company/smartticket/internal/service"
)

// ----------------------------------------------------------------------------
// Local output view
// ----------------------------------------------------------------------------
//
// servicemgmt.ServiceResponse cannot be reused as an MCP Output: its
// SupportChannels/Tags ([]string) and EscalationRules/Configuration (map)
// fields lack `omitempty`, so a nil value marshals to JSON null and the go-sdk
// rejects it against the inferred array/object output schema (breaking the
// success path on nil data and every error path, which returns a zero Out).
// svcResponse is the schema-safe MCP view: every slice/map/pointer field carries
// `omitempty`.
type svcResponse struct {
	ID              uint                   `json:"id" jsonschema:"the service's numeric ID"`
	ProductID       uint                   `json:"product_id" jsonschema:"the owning product's numeric ID"`
	Name            string                 `json:"name" jsonschema:"service name"`
	Code            string                 `json:"code" jsonschema:"unique service code"`
	Description     string                 `json:"description,omitempty" jsonschema:"service description"`
	Type            string                 `json:"type" jsonschema:"service type"`
	Status          string                 `json:"status" jsonschema:"service status"`
	Availability    string                 `json:"availability,omitempty" jsonschema:"service availability"`
	SupportChannels []string               `json:"support_channels,omitempty" jsonschema:"the service's support channels"`
	EscalationRules map[string]interface{} `json:"escalation_rules,omitempty" jsonschema:"escalation rules"`
	Configuration   map[string]interface{} `json:"configuration,omitempty" jsonschema:"service configuration"`
	Tags            []string               `json:"tags,omitempty" jsonschema:"service tags"`
	IsDeleted       bool                   `json:"is_deleted" jsonschema:"whether the service is soft-deleted"`
	CreatedAt       time.Time              `json:"created_at" jsonschema:"when the service was created"`
	UpdatedAt       time.Time              `json:"updated_at" jsonschema:"when the service was last updated"`
}

// svcResponseFrom converts a service-layer ServiceResponse into the schema-safe
// MCP view. The embedded Product association is dropped to keep the view flat.
func svcResponseFrom(r *servicemgmt.ServiceResponse) svcResponse {
	if r == nil {
		return svcResponse{}
	}
	return svcResponse{
		ID:              r.ID,
		ProductID:       r.ProductID,
		Name:            r.Name,
		Code:            r.Code,
		Description:     r.Description,
		Type:            r.Type,
		Status:          r.Status,
		Availability:    r.Availability,
		SupportChannels: r.SupportChannels,
		EscalationRules: r.EscalationRules,
		Configuration:   r.Configuration,
		Tags:            r.Tags,
		IsDeleted:       r.IsDeleted,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

// svcResponsesFrom converts a slice of service-layer responses into views.
func svcResponsesFrom(rs []servicemgmt.ServiceResponse) []svcResponse {
	if len(rs) == 0 {
		return nil
	}
	views := make([]svcResponse, len(rs))
	for i := range rs {
		views[i] = svcResponseFrom(&rs[i])
	}
	return views
}

// registerServiceTools registers the service-domain MCP tools. Each tool is a
// thin MCP-facing wrapper around a named svc* business function so the logic is
// unit-testable in isolation; cross-cutting concerns (RBAC, recover, logging,
// error mapping) are applied uniformly by registerTool.
//
// See server.go for the tool implementation conventions and auth_whoami template.
func registerServiceTools(s *mcp.Server, b Backend) {
	registerTool(s,
		"service_create",
		"Create a new service in the service catalog under a product.",
		"service:write",
		func(ctx context.Context, in svcCreateInput) (svcResponse, string, error) {
			return svcCreate(ctx, b, in)
		},
	)

	registerTool(s,
		"service_get",
		"Fetch a single service by its numeric ID.",
		"service:read",
		func(ctx context.Context, in svcGetInput) (svcResponse, string, error) {
			return svcGet(ctx, b, in)
		},
	)

	registerTool(s,
		"service_list",
		"List services with pagination and optional filtering.",
		"service:read",
		func(ctx context.Context, in svcListInput) (svcListOutput, string, error) {
			return svcList(ctx, b, in)
		},
	)

	registerTool(s,
		"service_update",
		"Update fields of an existing service.",
		"service:write",
		func(ctx context.Context, in svcUpdateInput) (svcResponse, string, error) {
			return svcUpdate(ctx, b, in)
		},
	)

	registerTool(s,
		"service_delete",
		"Soft-delete a service by its numeric ID.",
		"service:write",
		func(ctx context.Context, in svcDeleteInput) (svcActionOutput, string, error) {
			return svcDelete(ctx, b, in)
		},
	)

	registerTool(s,
		"service_activate",
		"Activate a service, setting its status to active.",
		"service:write",
		func(ctx context.Context, in svcActivateInput) (svcActionOutput, string, error) {
			return svcActivate(ctx, b, in)
		},
	)

	registerTool(s,
		"service_deactivate",
		"Deactivate a service, setting its status to inactive.",
		"service:write",
		func(ctx context.Context, in svcDeactivateInput) (svcActionOutput, string, error) {
			return svcDeactivate(ctx, b, in)
		},
	)
}

// ----------------------------------------------------------------------------
// Input / output schemas
// ----------------------------------------------------------------------------

// svcCreateInput is the MCP input schema for service_create.
type svcCreateInput struct {
	ProductID       uint   `json:"product_id" jsonschema:"numeric ID of the product this service belongs to"`
	Name            string `json:"name" jsonschema:"human-readable service name"`
	Code            string `json:"code" jsonschema:"unique service code (stored uppercased)"`
	Description     string `json:"description,omitempty" jsonschema:"optional description of the service"`
	Type            string `json:"type" jsonschema:"service type: infrastructure, application, support, or consulting"`
	Status          string `json:"status,omitempty" jsonschema:"optional status: active, inactive, or maintenance (defaults to active)"`
	Availability    string `json:"availability,omitempty" jsonschema:"optional availability: 24x7, business_hours, or custom (defaults to 24x7)"`
	SupportChannels string `json:"support_channels,omitempty" jsonschema:"optional JSON array of support channels"`
	EscalationRules string `json:"escalation_rules,omitempty" jsonschema:"optional JSON object describing escalation rules"`
	Configuration   string `json:"configuration,omitempty" jsonschema:"optional JSON object of service configuration"`
	Tags            string `json:"tags,omitempty" jsonschema:"optional JSON array of tags"`
}

// svcGetInput is the MCP input schema for service_get.
type svcGetInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the service to fetch"`
}

// svcListInput is the MCP input schema for service_list.
type svcListInput struct {
	Page      int    `json:"page,omitempty" jsonschema:"page number, 1-based (defaults to 1)"`
	PageSize  int    `json:"page_size,omitempty" jsonschema:"items per page, 1-100 (defaults to 20)"`
	Search    string `json:"search,omitempty" jsonschema:"optional text matched against name, code, and description"`
	ProductID uint   `json:"product_id,omitempty" jsonschema:"optional filter by product ID"`
	Type      string `json:"type,omitempty" jsonschema:"optional filter by service type"`
	Status    string `json:"status,omitempty" jsonschema:"optional filter by status"`
	SortBy    string `json:"sort_by,omitempty" jsonschema:"optional sort field (defaults to created_at)"`
	SortOrder string `json:"sort_order,omitempty" jsonschema:"optional sort order: asc or desc (defaults to desc)"`
}

// svcListOutput is the structured output of service_list.
type svcListOutput struct {
	Services []svcResponse `json:"services,omitempty" jsonschema:"the page of services matching the query"`
	Total    int64         `json:"total" jsonschema:"total number of services matching the filters"`
}

// svcUpdateInput is the MCP input schema for service_update. The ID identifies
// the target; the remaining fields are optional partial updates.
type svcUpdateInput struct {
	ID              uint   `json:"id" jsonschema:"numeric ID of the service to update"`
	ProductID       *uint  `json:"product_id,omitempty" jsonschema:"optional new owning product ID"`
	Name            string `json:"name,omitempty" jsonschema:"optional new service name"`
	Code            string `json:"code,omitempty" jsonschema:"optional new service code"`
	Description     string `json:"description,omitempty" jsonschema:"optional new description"`
	Type            string `json:"type,omitempty" jsonschema:"optional new service type"`
	Status          string `json:"status,omitempty" jsonschema:"optional new status"`
	Availability    string `json:"availability,omitempty" jsonschema:"optional new availability setting"`
	SupportChannels string `json:"support_channels,omitempty" jsonschema:"optional new JSON array of support channels"`
	EscalationRules string `json:"escalation_rules,omitempty" jsonschema:"optional new JSON object of escalation rules"`
	Configuration   string `json:"configuration,omitempty" jsonschema:"optional new JSON object of configuration"`
	Tags            string `json:"tags,omitempty" jsonschema:"optional new JSON array of tags"`
}

// svcDeleteInput is the MCP input schema for service_delete.
type svcDeleteInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the service to delete"`
}

// svcActivateInput is the MCP input schema for service_activate.
type svcActivateInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the service to activate"`
}

// svcDeactivateInput is the MCP input schema for service_deactivate.
type svcDeactivateInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the service to deactivate"`
}

// svcActionOutput is the structured output of mutating tools that do not return
// a service body (delete/activate/deactivate).
type svcActionOutput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the affected service"`
}

// ----------------------------------------------------------------------------
// Business functions (testable in isolation)
// ----------------------------------------------------------------------------

// svcCreate translates the MCP input into a service-layer request and creates a
// service via the Backend.
func svcCreate(_ context.Context, b Backend, in svcCreateInput) (svcResponse, string, error) {
	req := &servicemgmt.CreateServiceRequest{
		ProductID:       in.ProductID,
		Name:            in.Name,
		Code:            in.Code,
		Description:     in.Description,
		Type:            in.Type,
		Status:          in.Status,
		Availability:    in.Availability,
		SupportChannels: in.SupportChannels,
		EscalationRules: in.EscalationRules,
		Configuration:   in.Configuration,
		Tags:            in.Tags,
	}
	resp, err := b.CreateService(req)
	if err != nil {
		return svcResponse{}, "", err
	}
	return svcResponseFrom(resp), fmt.Sprintf("Created service #%d (%s).", resp.ID, resp.Name), nil
}

// svcGet fetches a single service by ID.
func svcGet(_ context.Context, b Backend, in svcGetInput) (svcResponse, string, error) {
	resp, err := b.GetService(in.ID)
	if err != nil {
		return svcResponse{}, "", err
	}
	return svcResponseFrom(resp), fmt.Sprintf("Service #%d (%s).", resp.ID, resp.Name), nil
}

// svcList lists services with pagination and filtering.
func svcList(_ context.Context, b Backend, in svcListInput) (svcListOutput, string, error) {
	req := &servicemgmt.ListServicesRequest{
		Page:      in.Page,
		PageSize:  in.PageSize,
		Search:    in.Search,
		ProductID: in.ProductID,
		Type:      in.Type,
		Status:    in.Status,
		SortBy:    in.SortBy,
		SortOrder: in.SortOrder,
	}
	services, total, err := b.ListServices(req)
	if err != nil {
		return svcListOutput{}, "", err
	}
	out := svcListOutput{Services: svcResponsesFrom(services), Total: total}
	return out, fmt.Sprintf("Listed %d of %d service(s).", len(services), total), nil
}

// svcUpdate applies a partial update to an existing service.
func svcUpdate(_ context.Context, b Backend, in svcUpdateInput) (svcResponse, string, error) {
	req := &servicemgmt.UpdateServiceRequest{
		ProductID:       in.ProductID,
		Name:            in.Name,
		Code:            in.Code,
		Description:     in.Description,
		Type:            in.Type,
		Status:          in.Status,
		Availability:    in.Availability,
		SupportChannels: in.SupportChannels,
		EscalationRules: in.EscalationRules,
		Configuration:   in.Configuration,
		Tags:            in.Tags,
	}
	resp, err := b.UpdateService(in.ID, req)
	if err != nil {
		return svcResponse{}, "", err
	}
	return svcResponseFrom(resp), fmt.Sprintf("Updated service #%d (%s).", resp.ID, resp.Name), nil
}

// svcDelete soft-deletes a service by ID.
func svcDelete(_ context.Context, b Backend, in svcDeleteInput) (svcActionOutput, string, error) {
	if err := b.DeleteService(in.ID); err != nil {
		return svcActionOutput{}, "", err
	}
	return svcActionOutput{ID: in.ID}, fmt.Sprintf("Deleted service #%d.", in.ID), nil
}

// svcActivate activates a service by ID.
func svcActivate(_ context.Context, b Backend, in svcActivateInput) (svcActionOutput, string, error) {
	if err := b.ActivateService(in.ID); err != nil {
		return svcActionOutput{}, "", err
	}
	return svcActionOutput{ID: in.ID}, fmt.Sprintf("Activated service #%d.", in.ID), nil
}

// svcDeactivate deactivates a service by ID.
func svcDeactivate(_ context.Context, b Backend, in svcDeactivateInput) (svcActionOutput, string, error) {
	if err := b.DeactivateService(in.ID); err != nil {
		return svcActionOutput{}, "", err
	}
	return svcActionOutput{ID: in.ID}, fmt.Sprintf("Deactivated service #%d.", in.ID), nil
}
