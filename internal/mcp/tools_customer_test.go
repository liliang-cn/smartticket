package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/company/smartticket/internal/customer"
)

func TestCustomerCreateTool(t *testing.T) {
	mb := &MockBackend{}
	mb.On("CreateCustomer", &customer.CreateCustomerRequest{Name: "Acme"}).
		Return(&customer.CustomerResponse{ID: 7, Name: "Acme"}, nil)

	out, summary, err := customerCreate(mb, customerCreateInput{Name: "Acme"})
	require.NoError(t, err)
	assert.Equal(t, uint(7), out.ID)
	assert.Equal(t, "Acme", out.Name)
	assert.Contains(t, summary, "#7")
	mb.AssertExpectations(t)
}

func TestCustomerListTool(t *testing.T) {
	mb := &MockBackend{}
	mb.On("ListCustomers", &customer.ListCustomersRequest{Page: 1, PageSize: 20}).
		Return([]customer.CustomerResponse{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}, 2, nil)

	out, _, err := customerList(mb, customerListInput{})
	require.NoError(t, err)
	assert.Len(t, out.Customers, 2)
	assert.Equal(t, int64(2), out.Total)
	mb.AssertExpectations(t)
}

func TestCustomerUsersTool(t *testing.T) {
	mb := &MockBackend{}
	mb.On("ListCustomerUsers", uint(3)).
		Return([]customer.CustomerUserResponse{{ID: 9, Email: "a@acme.com"}}, nil)

	out, _, err := customerUsers(mb, customerUsersInput{ID: 3})
	require.NoError(t, err)
	require.Len(t, out.Users, 1)
	assert.Equal(t, "a@acme.com", out.Users[0].Email)
	mb.AssertExpectations(t)
}
