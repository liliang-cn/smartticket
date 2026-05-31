package subscription

import (
	"fmt"
	"testing"
	"time"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestService(t *testing.T) (*Service, *models.Customer, *models.Product, *models.SLATemplate) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&models.Customer{},
		&models.Product{},
		&models.SLATemplate{},
		&models.Subscription{},
	))

	customer := &models.Customer{Name: "Acme Corp"}
	require.NoError(t, db.Create(customer).Error)

	product := &models.Product{Name: "LINSTOR", Code: "LINSTOR", Status: "active"}
	require.NoError(t, db.Create(product).Error)

	slaTemplate := &models.SLATemplate{Name: "Premium 24x7"}
	require.NoError(t, db.Create(slaTemplate).Error)

	return NewService(db), customer, product, slaTemplate
}

func isValidationErr(err error) bool {
	appErr, ok := apperrors.IsAppError(err)
	if !ok {
		return false
	}
	return appErr.Code == apperrors.ErrCodeValidation || appErr.Code == apperrors.ErrCodeInvalidInput
}

func TestCreateAppliesDefaults(t *testing.T) {
	svc, customer, product, _ := setupTestService(t)

	resp, err := svc.Create(&CreateSubscriptionRequest{
		CustomerID: customer.ID,
		ProductID:  product.ID,
		Plan:       "Standard",
	})
	require.NoError(t, err)
	require.Equal(t, "per_node", resp.BillingUnit)
	require.Equal(t, "annual", resp.BillingPeriod)
	require.Equal(t, "USD", resp.Currency)
	require.Equal(t, "active", resp.Status)
	require.Equal(t, 1, resp.NodeCount)
	require.Equal(t, 1, resp.TotalUnits)
	require.Equal(t, "Acme Corp", resp.CustomerName)
	require.Equal(t, "LINSTOR", resp.ProductName)
}

func TestCreateUnknownCustomer(t *testing.T) {
	svc, _, product, _ := setupTestService(t)

	_, err := svc.Create(&CreateSubscriptionRequest{
		CustomerID: 9999,
		ProductID:  product.ID,
	})
	require.Error(t, err)
	require.True(t, isValidationErr(err), "expected validation error, got %v", err)
}

func TestCreateInvalidBillingUnit(t *testing.T) {
	svc, customer, product, _ := setupTestService(t)

	_, err := svc.Create(&CreateSubscriptionRequest{
		CustomerID:  customer.ID,
		ProductID:   product.ID,
		BillingUnit: "per_galaxy",
	})
	require.Error(t, err)
	require.True(t, isValidationErr(err), "expected validation error, got %v", err)
}

func TestCreateWithSLAAndExpiry(t *testing.T) {
	svc, customer, product, slaTemplate := setupTestService(t)

	resp, err := svc.Create(&CreateSubscriptionRequest{
		CustomerID:    customer.ID,
		ProductID:     product.ID,
		SLATemplateID: &slaTemplate.ID,
		NodeCount:     5,
		StartsAt:      time.Now().Add(-48 * time.Hour),
		ExpiresAt:     time.Now().Add(-24 * time.Hour),
	})
	require.NoError(t, err)
	require.Equal(t, "Premium 24x7", resp.SLATemplateName)
	require.Equal(t, 5, resp.TotalUnits)
	require.True(t, resp.IsExpired)
}

func TestListFilterByCustomer(t *testing.T) {
	svc, customer, product, _ := setupTestService(t)

	other := &models.Customer{Name: "Other Inc"}
	require.NoError(t, svc.db.Create(other).Error)

	_, err := svc.Create(&CreateSubscriptionRequest{CustomerID: customer.ID, ProductID: product.ID})
	require.NoError(t, err)
	_, err = svc.Create(&CreateSubscriptionRequest{CustomerID: other.ID, ProductID: product.ID})
	require.NoError(t, err)

	list, total, err := svc.List(&ListSubscriptionsRequest{Page: 1, PageSize: 20, CustomerID: &customer.ID})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	require.Equal(t, customer.ID, list[0].CustomerID)
}

func TestUpdateNodeCountAndStatus(t *testing.T) {
	svc, customer, product, _ := setupTestService(t)

	created, err := svc.Create(&CreateSubscriptionRequest{CustomerID: customer.ID, ProductID: product.ID})
	require.NoError(t, err)

	newCount := 8
	newStatus := "cancelled"
	updated, err := svc.Update(created.ID, &UpdateSubscriptionRequest{
		NodeCount: &newCount,
		Status:    &newStatus,
	})
	require.NoError(t, err)
	require.Equal(t, 8, updated.NodeCount)
	require.Equal(t, 8, updated.TotalUnits)
	require.Equal(t, "cancelled", updated.Status)
}

func TestDelete(t *testing.T) {
	svc, customer, product, _ := setupTestService(t)

	created, err := svc.Create(&CreateSubscriptionRequest{CustomerID: customer.ID, ProductID: product.ID})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(created.ID))

	_, err = svc.Get(created.ID)
	require.Error(t, err)
}
