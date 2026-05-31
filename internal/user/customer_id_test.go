package user

import (
	"testing"
	"time"

	"github.com/company/smartticket/internal/auth"
	"github.com/company/smartticket/internal/database"
	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUserService(t *testing.T, db *database.Database) *Service {
	t.Helper()
	authRepo := auth.NewRepository(db.DB)
	authService := auth.NewService(db.DB, "test-secret", time.Hour, time.Hour*24, "test-issuer")
	return NewService(db.DB, authRepo, authService)
}

func TestCreateUser_CustomerRoleRequiresCustomerID(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := newUserService(t, db)
		createTestRole(t, db, "customer")

		_, err := service.CreateUser(&CreateUserRequest{
			Email:     "nocust@example.com",
			Username:  "nocust",
			FirstName: "No",
			LastName:  "Customer",
			Password:  "Password123!",
			Role:      "customer",
			IsActive:  true,
		})
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeValidation, appErr.Code)
		assert.Contains(t, appErr.Message, "customer_id")
	})
}

func TestCreateUser_TeamRoleForbidsCustomerID(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := newUserService(t, db)
		createTestRole(t, db, "engineer")

		// A customer exists, but team roles must still reject customer_id.
		c := &models.Customer{Name: "Acme", IsActive: true}
		require.NoError(t, db.DB.Create(c).Error)
		cid := c.ID

		_, err := service.CreateUser(&CreateUserRequest{
			Email:      "eng@example.com",
			Username:   "enguser",
			FirstName:  "Eng",
			LastName:   "User",
			Password:   "Password123!",
			Role:       "engineer",
			IsActive:   true,
			CustomerID: &cid,
		})
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeValidation, appErr.Code)
		assert.Contains(t, appErr.Message, "customer_id")
	})
}

func TestCreateUser_CustomerRoleWithValidCustomerID(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := newUserService(t, db)
		createTestRole(t, db, "customer")

		c := &models.Customer{Name: "Acme", IsActive: true}
		require.NoError(t, db.DB.Create(c).Error)
		cid := c.ID

		result, err := service.CreateUser(&CreateUserRequest{
			Email:      "cust@example.com",
			Username:   "custuser",
			FirstName:  "Cust",
			LastName:   "User",
			Password:   "Password123!",
			Role:       "customer",
			IsActive:   true,
			CustomerID: &cid,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify the CustomerID was persisted on the user record.
		var stored models.User
		require.NoError(t, db.DB.First(&stored, result.ID).Error)
		require.NotNil(t, stored.CustomerID)
		assert.Equal(t, cid, *stored.CustomerID)
		assert.Equal(t, "customer", stored.Role)
	})
}

func TestCreateUser_CustomerRoleWithUnknownCustomerID(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := newUserService(t, db)
		createTestRole(t, db, "customer")

		bogus := uint(424242)
		_, err := service.CreateUser(&CreateUserRequest{
			Email:      "ghost@example.com",
			Username:   "ghost",
			FirstName:  "Ghost",
			LastName:   "User",
			Password:   "Password123!",
			Role:       "customer",
			IsActive:   true,
			CustomerID: &bogus,
		})
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeValidation, appErr.Code)
	})
}
