package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/subscription"
)

// subscriptionView is the schema-safe MCP view of a subscription. Pointer fields
// carry omitempty so a nil value does not marshal to JSON null (which the go-sdk
// rejects against the inferred output schema).
type subscriptionView struct {
	ID              uint      `json:"id" jsonschema:"the subscription's numeric ID"`
	CustomerID      uint      `json:"customer_id" jsonschema:"owning customer ID"`
	CustomerName    string    `json:"customer_name,omitempty" jsonschema:"owning customer name"`
	ProductID       uint      `json:"product_id" jsonschema:"subscribed product ID"`
	ProductName     string    `json:"product_name,omitempty" jsonschema:"subscribed product name"`
	SLATemplateID   *uint     `json:"sla_template_id,omitempty" jsonschema:"associated SLA template ID, if any"`
	SLATemplateName string    `json:"sla_template_name,omitempty" jsonschema:"associated SLA template name"`
	Plan            string    `json:"plan,omitempty" jsonschema:"plan name/tier"`
	BillingUnit     string    `json:"billing_unit,omitempty" jsonschema:"per_node or per_cluster"`
	NodeCount       int       `json:"node_count" jsonschema:"licensed node count"`
	TotalUnits      int       `json:"total_units" jsonschema:"total billable units"`
	BillingPeriod   string    `json:"billing_period,omitempty" jsonschema:"annual or monthly"`
	StartsAt        time.Time `json:"starts_at" jsonschema:"subscription start time"`
	ExpiresAt       time.Time `json:"expires_at" jsonschema:"subscription expiry time"`
	Status          string    `json:"status" jsonschema:"active, expired, or cancelled"`
	UnitPrice       float64   `json:"unit_price" jsonschema:"price per unit"`
	Currency        string    `json:"currency,omitempty" jsonschema:"ISO currency code"`
	Notes           string    `json:"notes,omitempty" jsonschema:"free-text notes"`
	IsExpired       bool      `json:"is_expired" jsonschema:"whether the subscription has expired"`
	CreatedAt       time.Time `json:"created_at" jsonschema:"when the subscription was created"`
	UpdatedAt       time.Time `json:"updated_at" jsonschema:"when the subscription was last updated"`
}

func subscriptionViewFrom(r *subscription.SubscriptionResponse) subscriptionView {
	if r == nil {
		return subscriptionView{}
	}
	return subscriptionView{
		ID: r.ID, CustomerID: r.CustomerID, CustomerName: r.CustomerName,
		ProductID: r.ProductID, ProductName: r.ProductName,
		SLATemplateID: r.SLATemplateID, SLATemplateName: r.SLATemplateName,
		Plan: r.Plan, BillingUnit: r.BillingUnit, NodeCount: r.NodeCount,
		TotalUnits: r.TotalUnits, BillingPeriod: r.BillingPeriod,
		StartsAt: r.StartsAt, ExpiresAt: r.ExpiresAt, Status: r.Status,
		UnitPrice: r.UnitPrice, Currency: r.Currency, Notes: r.Notes,
		IsExpired: r.IsExpired, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func subscriptionViewsFrom(rs []subscription.SubscriptionResponse) []subscriptionView {
	if len(rs) == 0 {
		return nil
	}
	views := make([]subscriptionView, len(rs))
	for i := range rs {
		views[i] = subscriptionViewFrom(&rs[i])
	}
	return views
}

// registerSubscriptionTools registers the subscription-domain MCP tools.
// See server.go for the tool implementation conventions.
func registerSubscriptionTools(s *mcp.Server, b Backend) {
	registerTool(s, "subscription_create",
		"Create a subscription linking a customer to a product, with billing and SLA terms.",
		"subscription:write",
		func(ctx context.Context, in subscriptionCreateInput) (subscriptionView, string, error) {
			return subscriptionCreate(ctx, b, in)
		})

	registerTool(s, "subscription_get",
		"Fetch a single subscription by its numeric ID.",
		"subscription:read",
		func(ctx context.Context, in subscriptionIDInput) (subscriptionView, string, error) {
			return subscriptionGet(ctx, b, in)
		})

	registerTool(s, "subscription_list",
		"List subscriptions with pagination and optional filtering by customer or status.",
		"subscription:read",
		func(ctx context.Context, in subscriptionListInput) (subscriptionListOutput, string, error) {
			return subscriptionList(ctx, b, in)
		})

	registerTool(s, "subscription_update",
		"Update an existing subscription by ID. Only provided fields are changed.",
		"subscription:write",
		func(ctx context.Context, in subscriptionUpdateInput) (subscriptionView, string, error) {
			return subscriptionUpdate(ctx, b, in)
		})

	registerTool(s, "subscription_delete",
		"Soft-delete a subscription by its numeric ID.",
		"subscription:write",
		func(ctx context.Context, in subscriptionIDInput) (deleteOutput, string, error) {
			return subscriptionDelete(ctx, b, in)
		})
}

// ---- schemas ----

type subscriptionCreateInput struct {
	CustomerID    uint       `json:"customer_id" jsonschema:"owning customer ID (required)"`
	ProductID     uint       `json:"product_id" jsonschema:"subscribed product ID (required)"`
	SLATemplateID *uint      `json:"sla_template_id,omitempty" jsonschema:"SLA template ID to attach"`
	Plan          string     `json:"plan,omitempty" jsonschema:"plan name/tier"`
	BillingUnit   string     `json:"billing_unit,omitempty" jsonschema:"per_node or per_cluster"`
	NodeCount     int        `json:"node_count,omitempty" jsonschema:"licensed node count"`
	BillingPeriod string     `json:"billing_period,omitempty" jsonschema:"annual or monthly"`
	StartsAt      *time.Time `json:"starts_at,omitempty" jsonschema:"start time (RFC3339)"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty" jsonschema:"expiry time (RFC3339)"`
	Status        string     `json:"status,omitempty" jsonschema:"active, expired, or cancelled"`
	UnitPrice     float64    `json:"unit_price,omitempty" jsonschema:"price per unit"`
	Currency      string     `json:"currency,omitempty" jsonschema:"ISO currency code"`
	Notes         string     `json:"notes,omitempty" jsonschema:"free-text notes"`
}

type subscriptionIDInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the subscription"`
}

type subscriptionListInput struct {
	Page       int    `json:"page,omitempty" jsonschema:"page number, 1-based (default 1)"`
	PageSize   int    `json:"page_size,omitempty" jsonschema:"items per page, 1-100 (default 20)"`
	CustomerID *uint  `json:"customer_id,omitempty" jsonschema:"filter by customer ID"`
	Status     string `json:"status,omitempty" jsonschema:"filter by status"`
}

type subscriptionListOutput struct {
	Subscriptions []subscriptionView `json:"subscriptions,omitempty" jsonschema:"the page of subscriptions"`
	Total         int64              `json:"total" jsonschema:"total matching subscriptions"`
	Page          int                `json:"page" jsonschema:"page number returned"`
	PageSize      int                `json:"page_size" jsonschema:"page size used"`
}

type subscriptionUpdateInput struct {
	ID            uint       `json:"id" jsonschema:"numeric ID of the subscription to update"`
	SLATemplateID *uint      `json:"sla_template_id,omitempty" jsonschema:"new SLA template ID"`
	Plan          *string    `json:"plan,omitempty" jsonschema:"new plan"`
	BillingUnit   *string    `json:"billing_unit,omitempty" jsonschema:"per_node or per_cluster"`
	NodeCount     *int       `json:"node_count,omitempty" jsonschema:"new node count"`
	BillingPeriod *string    `json:"billing_period,omitempty" jsonschema:"annual or monthly"`
	StartsAt      *time.Time `json:"starts_at,omitempty" jsonschema:"new start time (RFC3339)"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty" jsonschema:"new expiry time (RFC3339)"`
	Status        *string    `json:"status,omitempty" jsonschema:"new status"`
	UnitPrice     *float64   `json:"unit_price,omitempty" jsonschema:"new unit price"`
	Currency      *string    `json:"currency,omitempty" jsonschema:"new currency code"`
	Notes         *string    `json:"notes,omitempty" jsonschema:"new notes"`
}

// ---- closures ----

func subscriptionCreate(_ context.Context, b Backend, in subscriptionCreateInput) (subscriptionView, string, error) {
	req := &subscription.CreateSubscriptionRequest{
		CustomerID: in.CustomerID, ProductID: in.ProductID, SLATemplateID: in.SLATemplateID,
		Plan: in.Plan, BillingUnit: in.BillingUnit, NodeCount: in.NodeCount,
		BillingPeriod: in.BillingPeriod, Status: in.Status, UnitPrice: in.UnitPrice,
		Currency: in.Currency, Notes: in.Notes,
	}
	if in.StartsAt != nil {
		req.StartsAt = *in.StartsAt
	}
	if in.ExpiresAt != nil {
		req.ExpiresAt = *in.ExpiresAt
	}
	resp, err := b.CreateSubscription(req)
	if err != nil {
		return subscriptionView{}, "", err
	}
	return subscriptionViewFrom(resp), fmt.Sprintf("Created subscription #%d for customer #%d.", resp.ID, resp.CustomerID), nil
}

func subscriptionGet(_ context.Context, b Backend, in subscriptionIDInput) (subscriptionView, string, error) {
	resp, err := b.GetSubscription(in.ID)
	if err != nil {
		return subscriptionView{}, "", err
	}
	return subscriptionViewFrom(resp), fmt.Sprintf("Subscription #%d (%s, status %s).", resp.ID, resp.ProductName, resp.Status), nil
}

func subscriptionList(_ context.Context, b Backend, in subscriptionListInput) (subscriptionListOutput, string, error) {
	req := &subscription.ListSubscriptionsRequest{
		Page: in.Page, PageSize: in.PageSize, CustomerID: in.CustomerID, Status: in.Status,
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}
	items, total, err := b.ListSubscriptions(req)
	if err != nil {
		return subscriptionListOutput{}, "", err
	}
	out := subscriptionListOutput{
		Subscriptions: subscriptionViewsFrom(items), Total: total, Page: req.Page, PageSize: req.PageSize,
	}
	return out, fmt.Sprintf("Returned %d of %d subscription(s).", len(items), total), nil
}

func subscriptionUpdate(_ context.Context, b Backend, in subscriptionUpdateInput) (subscriptionView, string, error) {
	req := &subscription.UpdateSubscriptionRequest{
		SLATemplateID: in.SLATemplateID, Plan: in.Plan, BillingUnit: in.BillingUnit,
		NodeCount: in.NodeCount, BillingPeriod: in.BillingPeriod, StartsAt: in.StartsAt,
		ExpiresAt: in.ExpiresAt, Status: in.Status, UnitPrice: in.UnitPrice,
		Currency: in.Currency, Notes: in.Notes,
	}
	resp, err := b.UpdateSubscription(in.ID, req)
	if err != nil {
		return subscriptionView{}, "", err
	}
	return subscriptionViewFrom(resp), fmt.Sprintf("Updated subscription #%d.", resp.ID), nil
}

func subscriptionDelete(_ context.Context, b Backend, in subscriptionIDInput) (deleteOutput, string, error) {
	if err := b.DeleteSubscription(in.ID); err != nil {
		return deleteOutput{}, "", err
	}
	return deleteOutput{ID: in.ID, Deleted: true}, fmt.Sprintf("Deleted subscription #%d.", in.ID), nil
}
