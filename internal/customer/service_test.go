package customer

import (
	"encoding/json"
	"testing"

	"github.com/company/smartticket/internal/database"
	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func marshalJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func TestCustomerService_CreateCustomer(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		req := &CreateCustomerRequest{
			Name:        "Acme Corp",
			Code:        "acme",
			Domain:      "Acme.com",
			Description: "Test customer",
		}

		result, err := service.CreateCustomer(req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotZero(t, result.ID)
		assert.Equal(t, "Acme Corp", result.Name)
		assert.Equal(t, "ACME", result.Code)       // normalized to upper
		assert.Equal(t, "acme.com", result.Domain) // normalized to lower
		assert.True(t, result.IsActive)            // defaults to active
	})
}

func TestCustomerService_CreateCustomer_RequiresName(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		_, err := service.CreateCustomer(&CreateCustomerRequest{Name: "   "})
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeInvalidInput, appErr.Code)
	})
}

func TestCustomerService_CreateCustomer_CodeConflict(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		_, err := service.CreateCustomer(&CreateCustomerRequest{Name: "First", Code: "DUP"})
		require.NoError(t, err)

		_, err = service.CreateCustomer(&CreateCustomerRequest{Name: "Second", Code: "dup"})
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeConflict, appErr.Code)
	})
}

// Code is optional: multiple customers without a code must coexist (the unique
// index stores NULL, not "", so they do not collide).
func TestCustomerService_CreateCustomer_NoCodeAllowsMany(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		_, err := service.CreateCustomer(&CreateCustomerRequest{Name: "No Code One"})
		require.NoError(t, err)
		_, err = service.CreateCustomer(&CreateCustomerRequest{Name: "No Code Two"})
		require.NoError(t, err)
	})
}

func TestCustomerService_GetCustomer(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		created, err := service.CreateCustomer(&CreateCustomerRequest{Name: "GetMe"})
		require.NoError(t, err)

		got, err := service.GetCustomer(created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "GetMe", got.Name)
	})
}

func TestCustomerService_GetCustomer_NotFound(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		_, err := service.GetCustomer(999999)
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeNotFound, appErr.Code)
	})
}

func TestCustomerService_ListCustomers(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		inactive := false
		_, err := service.CreateCustomer(&CreateCustomerRequest{Name: "Alpha Inc", Code: "ALPHA"})
		require.NoError(t, err)
		_, err = service.CreateCustomer(&CreateCustomerRequest{Name: "Beta Inc", Code: "BETA"})
		require.NoError(t, err)
		_, err = service.CreateCustomer(&CreateCustomerRequest{Name: "Gamma Co", Code: "GAMMA", IsActive: &inactive})
		require.NoError(t, err)

		// List all
		all, total, err := service.ListCustomers(&ListCustomersRequest{Page: 1, PageSize: 20})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, all, 3)

		// Search by name
		found, total, err := service.ListCustomers(&ListCustomersRequest{Page: 1, PageSize: 20, Search: "Beta"})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, found, 1)
		assert.Equal(t, "Beta Inc", found[0].Name)

		// Filter by is_active=true
		active := true
		activeList, total, err := service.ListCustomers(&ListCustomersRequest{Page: 1, PageSize: 20, IsActive: &active})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, activeList, 2)
	})
}

func TestCustomerService_UpdateCustomer(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		created, err := service.CreateCustomer(&CreateCustomerRequest{Name: "Old Name", Code: "OLD"})
		require.NoError(t, err)

		inactive := false
		updated, err := service.UpdateCustomer(created.ID, &UpdateCustomerRequest{
			Name:     "New Name",
			Code:     "new",
			IsActive: &inactive,
		})
		require.NoError(t, err)
		assert.Equal(t, "New Name", updated.Name)
		assert.Equal(t, "NEW", updated.Code)
		assert.False(t, updated.IsActive)
	})
}

func TestCustomerService_UpdateCustomer_CodeConflict(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		_, err := service.CreateCustomer(&CreateCustomerRequest{Name: "One", Code: "ONE"})
		require.NoError(t, err)
		two, err := service.CreateCustomer(&CreateCustomerRequest{Name: "Two", Code: "TWO"})
		require.NoError(t, err)

		_, err = service.UpdateCustomer(two.ID, &UpdateCustomerRequest{Code: "ONE"})
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeConflict, appErr.Code)
	})
}

func TestCustomerService_DeleteCustomer(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		created, err := service.CreateCustomer(&CreateCustomerRequest{Name: "DeleteMe"})
		require.NoError(t, err)

		require.NoError(t, service.DeleteCustomer(created.ID))

		// Not retrievable after soft delete.
		_, err = service.GetCustomer(created.ID)
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeNotFound, appErr.Code)

		// Row still present with soft-delete marker.
		var soft models.Customer
		require.NoError(t, db.DB.Unscoped().First(&soft, created.ID).Error)
		assert.True(t, soft.DeletedAt.Valid)
	})
}

func TestCustomerService_DeleteCustomer_NotFound(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		err := service.DeleteCustomer(999999)
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeNotFound, appErr.Code)
	})
}

func TestCustomerService_ListCustomerUsers_NoSensitiveFields(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		created, err := service.CreateCustomer(&CreateCustomerRequest{Name: "WithUsers", Code: "WITHUSERS"})
		require.NoError(t, err)

		cid := created.ID
		u1 := &models.User{
			Email:        "contact1@withusers.com",
			Username:     "contact1",
			FirstName:    "Contact",
			LastName:     "One",
			PasswordHash: "$2a$10$super.secret.hash.value.here",
			Role:         "customer",
			IsActive:     true,
			CustomerID:   &cid,
		}
		require.NoError(t, db.DB.Create(u1).Error)

		// A user belonging to a different customer must not be listed.
		other, err := service.CreateCustomer(&CreateCustomerRequest{Name: "Other", Code: "OTHER"})
		require.NoError(t, err)
		ocid := other.ID
		require.NoError(t, db.DB.Create(&models.User{
			Email:        "contact2@other.com",
			Username:     "contact2",
			PasswordHash: "$2a$10$another.secret.hash.value",
			Role:         "customer",
			IsActive:     true,
			CustomerID:   &ocid,
		}).Error)

		users, err := service.ListCustomerUsers(created.ID)
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, "contact1@withusers.com", users[0].Email)
		assert.Equal(t, "customer", users[0].Role)

		// CustomerUserResponse has no field that could carry the hash; serialize
		// to JSON and ensure the secret never leaks.
		assert.NotContains(t, marshalJSON(t, users), "super.secret")
	})
}

func TestCustomerService_ListCustomerUsers_NotFound(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := NewService(db.DB)

		_, err := service.ListCustomerUsers(999999)
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeNotFound, appErr.Code)
	})
}
