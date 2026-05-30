package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/product"
)

func TestProductCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:write"))

		in := productCreateInput{
			Name:         "Widget",
			Code:         "wid-1",
			Description:  "a widget",
			Category:     "hardware",
			Version:      "1.0",
			Status:       "active",
			IsManaged:    true,
			SupportLevel: "premium",
		}

		b.On("CreateProduct", mock.MatchedBy(func(req *product.CreateProductRequest) bool {
			return req.Name == "Widget" &&
				req.Code == "wid-1" &&
				req.Description == "a widget" &&
				req.Category == "hardware" &&
				req.Version == "1.0" &&
				req.Status == "active" &&
				req.IsManaged == true &&
				req.SupportLevel == "premium"
		})).Return(&product.ProductResponse{ID: 7, Name: "Widget", Code: "WID-1", Status: "active"}, nil)

		out, summary, err := productCreate(ctx, b, in)
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, uint(7), out.ID)
		assert.Contains(t, summary, "Widget")
		assert.Contains(t, summary, "#7")
		b.AssertExpectations(t)
	})

	t.Run("backend error", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:write"))

		b.On("CreateProduct", mock.Anything).
			Return(nil, apperrors.NewConflictError("产品代码已存在"))

		out, _, err := productCreate(ctx, b, productCreateInput{Name: "X", Code: "X"})
		require.Error(t, err)
		assert.Zero(t, out)
		b.AssertExpectations(t)
	})
}

func TestProductGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:read"))

		b.On("GetProduct", uint(42)).
			Return(&product.ProductResponse{ID: 42, Name: "Gizmo", Status: "active"}, nil)

		out, summary, err := productGet(ctx, b, productGetInput{ID: 42})
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, uint(42), out.ID)
		assert.Contains(t, summary, "Gizmo")
		b.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:read"))

		b.On("GetProduct", uint(99)).
			Return(nil, apperrors.NewNotFoundError("产品"))

		out, _, err := productGet(ctx, b, productGetInput{ID: 99})
		require.Error(t, err)
		assert.Zero(t, out)
		b.AssertExpectations(t)
	})
}

func TestProductList(t *testing.T) {
	t.Run("success with filters", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:read"))

		managed := true
		in := productListInput{
			Page:      2,
			PageSize:  10,
			Search:    "wid",
			Category:  "hardware",
			Status:    "active",
			IsManaged: &managed,
			SortBy:    "name",
			SortOrder: "asc",
		}

		b.On("ListProducts", mock.MatchedBy(func(req *product.ListProductsRequest) bool {
			return req.Page == 2 &&
				req.PageSize == 10 &&
				req.Search == "wid" &&
				req.Category == "hardware" &&
				req.Status == "active" &&
				req.IsManaged != nil && *req.IsManaged == true &&
				req.SortBy == "name" &&
				req.SortOrder == "asc"
		})).Return([]product.ProductResponse{
			{ID: 1, Name: "A"},
			{ID: 2, Name: "B"},
		}, 5, nil)

		out, summary, err := productList(ctx, b, in)
		require.NoError(t, err)
		assert.Len(t, out.Products, 2)
		assert.Equal(t, int64(5), out.Total)
		assert.Equal(t, 2, out.Page)
		assert.Equal(t, 10, out.PageSize)
		assert.Contains(t, summary, "2 of 5")
		b.AssertExpectations(t)
	})

	t.Run("backend error", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:read"))

		b.On("ListProducts", mock.Anything).
			Return(nil, 0, apperrors.NewInternalError("db down", nil))

		out, _, err := productList(ctx, b, productListInput{})
		require.Error(t, err)
		assert.Empty(t, out.Products)
		b.AssertExpectations(t)
	})
}

func TestProductUpdate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:write"))

		managed := false
		in := productUpdateInput{
			ID:        3,
			Name:      "Renamed",
			Status:    "deprecated",
			IsManaged: &managed,
		}

		b.On("UpdateProduct", uint(3), mock.MatchedBy(func(req *product.UpdateProductRequest) bool {
			return req.Name == "Renamed" &&
				req.Status == "deprecated" &&
				req.IsManaged != nil && *req.IsManaged == false
		})).Return(&product.ProductResponse{ID: 3, Name: "Renamed", Status: "deprecated"}, nil)

		out, summary, err := productUpdate(ctx, b, in)
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, "Renamed", out.Name)
		assert.Contains(t, summary, "#3")
		b.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:write"))

		b.On("UpdateProduct", uint(404), mock.Anything).
			Return(nil, apperrors.NewNotFoundError("产品"))

		out, _, err := productUpdate(ctx, b, productUpdateInput{ID: 404, Name: "x"})
		require.Error(t, err)
		assert.Zero(t, out)
		b.AssertExpectations(t)
	})
}

func TestProductDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:write"))

		b.On("DeleteProduct", uint(5)).Return(nil)

		out, summary, err := productDelete(ctx, b, productDeleteInput{ID: 5})
		require.NoError(t, err)
		assert.Equal(t, uint(5), out.ID)
		assert.True(t, out.Deleted)
		assert.Contains(t, summary, "#5")
		b.AssertExpectations(t)
	})

	t.Run("backend error", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:write"))

		b.On("DeleteProduct", uint(5)).
			Return(apperrors.NewBusinessRuleError("无法删除产品", "有关联服务"))

		out, _, err := productDelete(ctx, b, productDeleteInput{ID: 5})
		require.Error(t, err)
		assert.False(t, out.Deleted)
		b.AssertExpectations(t)
	})
}

func TestProductActivate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:write"))

		b.On("ActivateProduct", uint(8)).Return(nil)

		out, summary, err := productActivate(ctx, b, productActivateInput{ID: 8})
		require.NoError(t, err)
		assert.Equal(t, uint(8), out.ID)
		assert.Equal(t, "active", out.Status)
		assert.Contains(t, summary, "Activated")
		b.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:write"))

		b.On("ActivateProduct", uint(8)).
			Return(apperrors.NewNotFoundError("产品"))

		out, _, err := productActivate(ctx, b, productActivateInput{ID: 8})
		require.Error(t, err)
		assert.Empty(t, out.Status)
		b.AssertExpectations(t)
	})
}

func TestProductDeactivate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:write"))

		b.On("DeactivateProduct", uint(9)).Return(nil)

		out, summary, err := productDeactivate(ctx, b, productDeactivateInput{ID: 9})
		require.NoError(t, err)
		assert.Equal(t, uint(9), out.ID)
		assert.Equal(t, "inactive", out.Status)
		assert.Contains(t, summary, "Deactivated")
		b.AssertExpectations(t)
	})

	t.Run("backend error", func(t *testing.T) {
		b := new(MockBackend)
		ctx := ctxWithSession(newTestSession("product:write"))

		b.On("DeactivateProduct", uint(9)).
			Return(apperrors.NewInternalError("boom", nil))

		out, _, err := productDeactivate(ctx, b, productDeactivateInput{ID: 9})
		require.Error(t, err)
		assert.Empty(t, out.Status)
		b.AssertExpectations(t)
	})
}

// TestProductRegisterTools ensures the registration wiring compiles and runs
// against a real server instance without panicking.
func TestProductRegisterTools(t *testing.T) {
	b := new(MockBackend)
	s := NewMCPServer(b, []string{"product"})
	require.NotNil(t, s)
}
