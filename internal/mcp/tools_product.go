package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/product"
)

// ----------------------------------------------------------------------------
// Local output view
// ----------------------------------------------------------------------------
//
// product.ProductResponse cannot be reused as an MCP Output: its Configuration
// (map) and Tags ([]string) fields lack `omitempty`, so a nil value marshals to
// JSON null and the go-sdk rejects it against the inferred object/array output
// schema (breaking the success path on nil data and every error path, which
// returns a zero Out). productResponse is the schema-safe MCP view: every
// slice/map/pointer field carries `omitempty`.
type productResponse struct {
	ID            uint                   `json:"id" jsonschema:"the product's numeric ID"`
	Name          string                 `json:"name" jsonschema:"product display name"`
	Code          string                 `json:"code" jsonschema:"unique product code"`
	Description   string                 `json:"description,omitempty" jsonschema:"product description"`
	Category      string                 `json:"category,omitempty" jsonschema:"product category"`
	Version       string                 `json:"version,omitempty" jsonschema:"product version string"`
	Status        string                 `json:"status" jsonschema:"product status"`
	IsManaged     bool                   `json:"is_managed" jsonschema:"whether the product is actively managed"`
	SupportLevel  string                 `json:"support_level,omitempty" jsonschema:"product support level"`
	Documentation string                 `json:"documentation,omitempty" jsonschema:"documentation link or text"`
	Configuration map[string]interface{} `json:"configuration,omitempty" jsonschema:"product configuration"`
	Tags          []string               `json:"tags,omitempty" jsonschema:"product tags"`
	IsDeleted     bool                   `json:"is_deleted" jsonschema:"whether the product is soft-deleted"`
	CreatedAt     time.Time              `json:"created_at" jsonschema:"when the product was created"`
	UpdatedAt     time.Time              `json:"updated_at" jsonschema:"when the product was last updated"`
}

// productResponseFrom converts a service-layer product.ProductResponse into the
// schema-safe MCP view. The embedded Services association is dropped to keep the
// view flat.
func productResponseFrom(r *product.ProductResponse) productResponse {
	if r == nil {
		return productResponse{}
	}
	return productResponse{
		ID:            r.ID,
		Name:          r.Name,
		Code:          r.Code,
		Description:   r.Description,
		Category:      r.Category,
		Version:       r.Version,
		Status:        r.Status,
		IsManaged:     r.IsManaged,
		SupportLevel:  r.SupportLevel,
		Documentation: r.Documentation,
		Configuration: r.Configuration,
		Tags:          r.Tags,
		IsDeleted:     r.IsDeleted,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

// productResponsesFrom converts a slice of service-layer responses into views.
func productResponsesFrom(rs []product.ProductResponse) []productResponse {
	if len(rs) == 0 {
		return nil
	}
	views := make([]productResponse, len(rs))
	for i := range rs {
		views[i] = productResponseFrom(&rs[i])
	}
	return views
}

// registerProductTools registers the product-domain MCP tools.
// See server.go for the tool implementation conventions and auth_whoami template.
//
// All identifiers in this file are prefixed with "product" to avoid collisions
// with sibling domain files in the same package. The structured outputs use the
// MCP-local productResponse view (see above); the inputs are likewise
// MCP-specific structs translated into the service-layer DTOs.
func registerProductTools(s *mcp.Server, b Backend) {
	registerTool(s,
		"product_create",
		"Create a new product in the service catalog.",
		"product:write",
		func(ctx context.Context, in productCreateInput) (productResponse, string, error) {
			return productCreate(ctx, b, in)
		},
	)

	registerTool(s,
		"product_get",
		"Fetch a single product by its numeric ID, including associated services.",
		"product:read",
		func(ctx context.Context, in productGetInput) (productResponse, string, error) {
			return productGet(ctx, b, in)
		},
	)

	registerTool(s,
		"product_list",
		"List products with pagination, search, and filtering by category, status, or managed flag.",
		"product:read",
		func(ctx context.Context, in productListInput) (productListOutput, string, error) {
			return productList(ctx, b, in)
		},
	)

	registerTool(s,
		"product_update",
		"Update an existing product's fields by its numeric ID. Only provided fields are changed.",
		"product:write",
		func(ctx context.Context, in productUpdateInput) (productResponse, string, error) {
			return productUpdate(ctx, b, in)
		},
	)

	registerTool(s,
		"product_delete",
		"Soft-delete a product by its numeric ID. Fails if the product still has associated services or tickets.",
		"product:write",
		func(ctx context.Context, in productDeleteInput) (productDeleteOutput, string, error) {
			return productDelete(ctx, b, in)
		},
	)

	registerTool(s,
		"product_activate",
		"Activate a product by its numeric ID (sets status to active).",
		"product:write",
		func(ctx context.Context, in productActivateInput) (productStatusOutput, string, error) {
			return productActivate(ctx, b, in)
		},
	)

	registerTool(s,
		"product_deactivate",
		"Deactivate a product by its numeric ID (sets status to inactive).",
		"product:write",
		func(ctx context.Context, in productDeactivateInput) (productStatusOutput, string, error) {
			return productDeactivate(ctx, b, in)
		},
	)
}

// ----------------------------------------------------------------------------
// Input / Output schemas
// ----------------------------------------------------------------------------

// productCreateInput is the MCP input schema for product_create. It mirrors the
// fields of product.CreateProductRequest.
type productCreateInput struct {
	Name          string `json:"name" jsonschema:"product display name (required)"`
	Code          string `json:"code" jsonschema:"unique product code (required); normalized to upper-case"`
	Description   string `json:"description,omitempty" jsonschema:"free-text description"`
	Category      string `json:"category,omitempty" jsonschema:"product category"`
	Version       string `json:"version,omitempty" jsonschema:"product version string"`
	Status        string `json:"status,omitempty" jsonschema:"product status: active, inactive, or deprecated (defaults to active)"`
	IsManaged     bool   `json:"is_managed,omitempty" jsonschema:"whether the product is actively managed"`
	SupportLevel  string `json:"support_level,omitempty" jsonschema:"support level: basic, premium, or enterprise"`
	Documentation string `json:"documentation,omitempty" jsonschema:"documentation link or text"`
	Configuration string `json:"configuration,omitempty" jsonschema:"configuration as a JSON-encoded string"`
	Tags          string `json:"tags,omitempty" jsonschema:"tags as a JSON-encoded string array"`
}

// productGetInput is the MCP input schema for product_get.
type productGetInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the product to fetch"`
}

// productListInput is the MCP input schema for product_list. It exposes pagination
// and the supported filters.
type productListInput struct {
	Page      int    `json:"page,omitempty" jsonschema:"page number, 1-based (default 1)"`
	PageSize  int    `json:"page_size,omitempty" jsonschema:"number of items per page, 1-100 (default 20)"`
	Search    string `json:"search,omitempty" jsonschema:"search term matched against name, code, and description"`
	Category  string `json:"category,omitempty" jsonschema:"filter by exact category"`
	Status    string `json:"status,omitempty" jsonschema:"filter by status: active, inactive, or deprecated"`
	IsManaged *bool  `json:"is_managed,omitempty" jsonschema:"filter by managed flag; omit to include both"`
	SortBy    string `json:"sort_by,omitempty" jsonschema:"sort field: name, code, category, status, created_at, or updated_at (default created_at)"`
	SortOrder string `json:"sort_order,omitempty" jsonschema:"sort direction: asc or desc (default desc)"`
}

// productListOutput is the structured output of product_list.
type productListOutput struct {
	Products []productResponse `json:"products,omitempty" jsonschema:"the page of products"`
	Total    int64             `json:"total" jsonschema:"total number of products matching the filters"`
	Page     int               `json:"page" jsonschema:"the page number returned"`
	PageSize int               `json:"page_size" jsonschema:"the page size used"`
}

// productUpdateInput is the MCP input schema for product_update. All fields except
// ID are optional; only non-zero values are applied. IsManaged is a pointer so an
// explicit false can be distinguished from "unset".
type productUpdateInput struct {
	ID            uint   `json:"id" jsonschema:"numeric ID of the product to update"`
	Name          string `json:"name,omitempty" jsonschema:"new product display name"`
	Code          string `json:"code,omitempty" jsonschema:"new unique product code; normalized to upper-case"`
	Description   string `json:"description,omitempty" jsonschema:"new description"`
	Category      string `json:"category,omitempty" jsonschema:"new category"`
	Version       string `json:"version,omitempty" jsonschema:"new version string"`
	Status        string `json:"status,omitempty" jsonschema:"new status: active, inactive, or deprecated"`
	IsManaged     *bool  `json:"is_managed,omitempty" jsonschema:"new managed flag"`
	SupportLevel  string `json:"support_level,omitempty" jsonschema:"new support level: basic, premium, or enterprise"`
	Documentation string `json:"documentation,omitempty" jsonschema:"new documentation link or text"`
	Configuration string `json:"configuration,omitempty" jsonschema:"new configuration as a JSON-encoded string"`
	Tags          string `json:"tags,omitempty" jsonschema:"new tags as a JSON-encoded string array"`
}

// productDeleteInput is the MCP input schema for product_delete.
type productDeleteInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the product to delete"`
}

// productDeleteOutput reports the outcome of a product_delete call.
type productDeleteOutput struct {
	ID      uint `json:"id" jsonschema:"the ID of the deleted product"`
	Deleted bool `json:"deleted" jsonschema:"true when the product was deleted"`
}

// productActivateInput is the MCP input schema for product_activate.
type productActivateInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the product to activate"`
}

// productDeactivateInput is the MCP input schema for product_deactivate.
type productDeactivateInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the product to deactivate"`
}

// productStatusOutput reports the resulting status of an activate/deactivate call.
type productStatusOutput struct {
	ID     uint   `json:"id" jsonschema:"the product ID"`
	Status string `json:"status" jsonschema:"the product's status after the operation"`
}

// ----------------------------------------------------------------------------
// Business closures (named functions for direct unit testing)
// ----------------------------------------------------------------------------

// productCreate translates the MCP input into a service request and creates the
// product via the Backend.
func productCreate(_ context.Context, b Backend, in productCreateInput) (productResponse, string, error) {
	req := &product.CreateProductRequest{
		Name:          in.Name,
		Code:          in.Code,
		Description:   in.Description,
		Category:      in.Category,
		Version:       in.Version,
		Status:        in.Status,
		IsManaged:     in.IsManaged,
		SupportLevel:  in.SupportLevel,
		Documentation: in.Documentation,
		Configuration: in.Configuration,
		Tags:          in.Tags,
	}

	resp, err := b.CreateProduct(req)
	if err != nil {
		return productResponse{}, "", err
	}
	summary := fmt.Sprintf("Created product %q (#%d, code %s).", resp.Name, resp.ID, resp.Code)
	return productResponseFrom(resp), summary, nil
}

// productGet fetches a single product by ID.
func productGet(_ context.Context, b Backend, in productGetInput) (productResponse, string, error) {
	resp, err := b.GetProduct(in.ID)
	if err != nil {
		return productResponse{}, "", err
	}
	summary := fmt.Sprintf("Product %q (#%d, status %s).", resp.Name, resp.ID, resp.Status)
	return productResponseFrom(resp), summary, nil
}

// productList lists products with pagination and filtering.
func productList(_ context.Context, b Backend, in productListInput) (productListOutput, string, error) {
	req := &product.ListProductsRequest{
		Page:      in.Page,
		PageSize:  in.PageSize,
		Search:    in.Search,
		Category:  in.Category,
		Status:    in.Status,
		IsManaged: in.IsManaged,
		SortBy:    in.SortBy,
		SortOrder: in.SortOrder,
	}

	products, total, err := b.ListProducts(req)
	if err != nil {
		return productListOutput{}, "", err
	}

	out := productListOutput{
		Products: productResponsesFrom(products),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	summary := fmt.Sprintf("Returned %d of %d product(s).", len(products), total)
	return out, summary, nil
}

// productUpdate applies the provided fields to an existing product.
func productUpdate(_ context.Context, b Backend, in productUpdateInput) (productResponse, string, error) {
	req := &product.UpdateProductRequest{
		Name:          in.Name,
		Code:          in.Code,
		Description:   in.Description,
		Category:      in.Category,
		Version:       in.Version,
		Status:        in.Status,
		IsManaged:     in.IsManaged,
		SupportLevel:  in.SupportLevel,
		Documentation: in.Documentation,
		Configuration: in.Configuration,
		Tags:          in.Tags,
	}

	resp, err := b.UpdateProduct(in.ID, req)
	if err != nil {
		return productResponse{}, "", err
	}
	summary := fmt.Sprintf("Updated product %q (#%d).", resp.Name, resp.ID)
	return productResponseFrom(resp), summary, nil
}

// productDelete soft-deletes a product.
func productDelete(_ context.Context, b Backend, in productDeleteInput) (productDeleteOutput, string, error) {
	if err := b.DeleteProduct(in.ID); err != nil {
		return productDeleteOutput{}, "", err
	}
	out := productDeleteOutput{ID: in.ID, Deleted: true}
	summary := fmt.Sprintf("Deleted product #%d.", in.ID)
	return out, summary, nil
}

// productActivate activates a product.
func productActivate(_ context.Context, b Backend, in productActivateInput) (productStatusOutput, string, error) {
	if err := b.ActivateProduct(in.ID); err != nil {
		return productStatusOutput{}, "", err
	}
	out := productStatusOutput{ID: in.ID, Status: "active"}
	summary := fmt.Sprintf("Activated product #%d.", in.ID)
	return out, summary, nil
}

// productDeactivate deactivates a product.
func productDeactivate(_ context.Context, b Backend, in productDeactivateInput) (productStatusOutput, string, error) {
	if err := b.DeactivateProduct(in.ID); err != nil {
		return productStatusOutput{}, "", err
	}
	out := productStatusOutput{ID: in.ID, Status: "inactive"}
	summary := fmt.Sprintf("Deactivated product #%d.", in.ID)
	return out, summary, nil
}
