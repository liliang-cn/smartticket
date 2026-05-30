package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/customer"
)

// Customer management tools (team-only; gated by customer:read / customer:write).
// customer.CustomerResponse and CustomerUserResponse are flat scalar structs
// (no slice/map/model fields), so they are reused directly as tool Output; the
// list outputs wrap a slice with `omitempty` so a nil result omits the field
// rather than emitting JSON null.

// ----------------------------------------------------------------------------
// Inputs
// ----------------------------------------------------------------------------

type customerCreateInput struct {
	Name        string `json:"name" jsonschema:"the customer organization name (required)"`
	Code        string `json:"code,omitempty" jsonschema:"optional unique short code"`
	Domain      string `json:"domain,omitempty" jsonschema:"optional email domain, e.g. acme.com"`
	Description string `json:"description,omitempty" jsonschema:"optional description"`
	IsActive    *bool  `json:"is_active,omitempty" jsonschema:"whether the customer is active (defaults true)"`
}

type customerGetInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the customer to fetch"`
}

type customerListInput struct {
	Page     int    `json:"page,omitempty" jsonschema:"1-based page number (defaults to 1)"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"page size, max 100 (defaults to 20)"`
	Search   string `json:"search,omitempty" jsonschema:"free-text search over name/code/domain"`
	IsActive *bool  `json:"is_active,omitempty" jsonschema:"filter by active status"`
}

type customerUpdateInput struct {
	ID          uint   `json:"id" jsonschema:"the numeric ID of the customer to update"`
	Name        string `json:"name,omitempty" jsonschema:"new name"`
	Code        string `json:"code,omitempty" jsonschema:"new unique short code"`
	Domain      string `json:"domain,omitempty" jsonschema:"new email domain"`
	Description string `json:"description,omitempty" jsonschema:"new description"`
	IsActive    *bool  `json:"is_active,omitempty" jsonschema:"set active status"`
}

type customerDeleteInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the customer to delete"`
}

type customerUsersInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the customer whose contacts to list"`
}

// ----------------------------------------------------------------------------
// Outputs
// ----------------------------------------------------------------------------

type customerListOutput struct {
	Customers []customer.CustomerResponse `json:"customers,omitempty" jsonschema:"the matching customers"`
	Total     int64                       `json:"total" jsonschema:"total number of matching customers"`
	Page      int                         `json:"page" jsonschema:"the current page"`
	PageSize  int                         `json:"page_size" jsonschema:"the page size"`
}

type customerDeleteOutput struct {
	ID      uint `json:"id" jsonschema:"the numeric ID of the deleted customer"`
	Deleted bool `json:"deleted" jsonschema:"whether the customer was deleted"`
}

type customerUsersOutput struct {
	Users []customer.CustomerUserResponse `json:"users,omitempty" jsonschema:"the customer's contact users"`
}

// ----------------------------------------------------------------------------
// Registration
// ----------------------------------------------------------------------------

func registerCustomerTools(s *mcp.Server, b Backend) {
	registerTool(s, "customer_create", "Create a new customer organization.", "customer:write",
		func(_ context.Context, in customerCreateInput) (customer.CustomerResponse, string, error) {
			return customerCreate(b, in)
		})
	registerTool(s, "customer_get", "Fetch a single customer organization by numeric ID.", "customer:read",
		func(_ context.Context, in customerGetInput) (customer.CustomerResponse, string, error) {
			return customerGet(b, in)
		})
	registerTool(s, "customer_list", "List customer organizations with pagination, search, and active filter.", "customer:read",
		func(_ context.Context, in customerListInput) (customerListOutput, string, error) {
			return customerList(b, in)
		})
	registerTool(s, "customer_update", "Update a customer organization by numeric ID.", "customer:write",
		func(_ context.Context, in customerUpdateInput) (customer.CustomerResponse, string, error) {
			return customerUpdate(b, in)
		})
	registerTool(s, "customer_delete", "Delete (soft) a customer organization by numeric ID.", "customer:write",
		func(_ context.Context, in customerDeleteInput) (customerDeleteOutput, string, error) {
			return customerDelete(b, in)
		})
	registerTool(s, "customer_users", "List the contact users belonging to a customer organization.", "customer:read",
		func(_ context.Context, in customerUsersInput) (customerUsersOutput, string, error) {
			return customerUsers(b, in)
		})
}

// ----------------------------------------------------------------------------
// Handlers
// ----------------------------------------------------------------------------

func customerCreate(b Backend, in customerCreateInput) (customer.CustomerResponse, string, error) {
	resp, err := b.CreateCustomer(&customer.CreateCustomerRequest{
		Name: in.Name, Code: in.Code, Domain: in.Domain, Description: in.Description, IsActive: in.IsActive,
	})
	if err != nil {
		return customer.CustomerResponse{}, "", err
	}
	return *resp, fmt.Sprintf("created customer #%d (%s)", resp.ID, resp.Name), nil
}

func customerGet(b Backend, in customerGetInput) (customer.CustomerResponse, string, error) {
	resp, err := b.GetCustomer(in.ID)
	if err != nil {
		return customer.CustomerResponse{}, "", err
	}
	return *resp, fmt.Sprintf("fetched customer #%d (%s)", resp.ID, resp.Name), nil
}

func customerList(b Backend, in customerListInput) (customerListOutput, string, error) {
	req := &customer.ListCustomersRequest{Page: in.Page, PageSize: in.PageSize, Search: in.Search, IsActive: in.IsActive}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	list, total, err := b.ListCustomers(req)
	if err != nil {
		return customerListOutput{}, "", err
	}
	return customerListOutput{Customers: list, Total: total, Page: req.Page, PageSize: req.PageSize},
		fmt.Sprintf("listed %d of %d customer(s)", len(list), total), nil
}

func customerUpdate(b Backend, in customerUpdateInput) (customer.CustomerResponse, string, error) {
	resp, err := b.UpdateCustomer(in.ID, &customer.UpdateCustomerRequest{
		Name: in.Name, Code: in.Code, Domain: in.Domain, Description: in.Description, IsActive: in.IsActive,
	})
	if err != nil {
		return customer.CustomerResponse{}, "", err
	}
	return *resp, fmt.Sprintf("updated customer #%d", resp.ID), nil
}

func customerDelete(b Backend, in customerDeleteInput) (customerDeleteOutput, string, error) {
	if err := b.DeleteCustomer(in.ID); err != nil {
		return customerDeleteOutput{}, "", err
	}
	return customerDeleteOutput{ID: in.ID, Deleted: true}, fmt.Sprintf("deleted customer #%d", in.ID), nil
}

func customerUsers(b Backend, in customerUsersInput) (customerUsersOutput, string, error) {
	users, err := b.ListCustomerUsers(in.ID)
	if err != nil {
		return customerUsersOutput{}, "", err
	}
	return customerUsersOutput{Users: users}, fmt.Sprintf("listed %d contact user(s) for customer #%d", len(users), in.ID), nil
}
