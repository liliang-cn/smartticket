package user

import (
	"testing"

	"github.com/company/smartticket/internal/database"
	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These verify that user-input failures are classified as 400 (ValidationError)
// or 409 (Conflict) rather than leaking as 500 (INTERNAL_ERROR).

func TestCreateUser_WeakPasswordIsValidationError(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := newUserService(t, db)
		createTestRole(t, db, "engineer")

		_, err := service.CreateUser(&CreateUserRequest{
			Email:     "weak@example.com",
			Username:  "weakpw",
			FirstName: "Weak",
			LastName:  "Pw",
			Password:  "alllowercase", // no uppercase/digit/special
			Role:      "engineer",
			IsActive:  true,
		})
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeValidation, appErr.Code)
	})
}

func TestCreateUser_BadEmailAndUsernameAreValidationErrors(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := newUserService(t, db)
		createTestRole(t, db, "engineer")

		_, err := service.CreateUser(&CreateUserRequest{
			Email: "not-an-email", Username: "ok", FirstName: "A", LastName: "B",
			Password: "Password123!", Role: "engineer", IsActive: true,
		})
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeValidation, appErr.Code)

		_, err = service.CreateUser(&CreateUserRequest{
			Email: "good@example.com", Username: "bad name!", FirstName: "A", LastName: "B",
			Password: "Password123!", Role: "engineer", IsActive: true,
		})
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeValidation, appErr.Code)
	})
}

func TestCreateUser_DuplicateEmailIsConflict(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		service := newUserService(t, db)
		createTestRole(t, db, "engineer")

		req := &CreateUserRequest{
			Email: "dup@example.com", Username: "dupone", FirstName: "A", LastName: "B",
			Password: "Password123!", Role: "engineer", IsActive: true,
		}
		_, err := service.CreateUser(req)
		require.NoError(t, err)

		dup := *req
		dup.Username = "duptwo"
		_, err = service.CreateUser(&dup)
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeConflict, appErr.Code)
	})
}
