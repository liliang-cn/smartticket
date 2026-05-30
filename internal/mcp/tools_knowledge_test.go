package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/knowledge"
)

func TestKnowledgeCreate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("knowledge:write"))

	in := knowledgeCreateInput{
		Title:    "How to reset password",
		Content:  "Follow these steps to reset your password.",
		Category: "guide",
		Status:   "published",
	}
	want := &knowledge.KnowledgeArticleResponse{ID: 42, Title: in.Title, Status: "published"}

	mb.On("CreateKnowledgeArticle", uint(1), mock.MatchedBy(func(req *knowledge.CreateKnowledgeArticleRequest) bool {
		return req.Title == in.Title && req.Content == in.Content && req.Category == "guide" && req.Status == "published"
	})).Return(want, nil)

	out, summary, err := knowledgeCreate(ctx, mb, in)

	assert.NoError(t, err)
	assert.Equal(t, knowledgeArticleFrom(want), out)
	assert.Equal(t, uint(42), out.ID)
	assert.Equal(t, "published", out.Status)
	assert.Contains(t, summary, "#42")
	mb.AssertExpectations(t)
}

func TestKnowledgeCreateUnauthenticated(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(nil)

	out, _, err := knowledgeCreate(ctx, mb, knowledgeCreateInput{Title: "x"})

	assert.ErrorIs(t, err, ErrUnauthenticated)
	assert.Equal(t, knowledgeArticle{}, out)
	mb.AssertExpectations(t)
}

func TestKnowledgeGet(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("knowledge:read"))

	want := &knowledge.KnowledgeArticleResponse{ID: 7, Title: "Networking FAQ", Status: "published", ViewCount: 12}
	mb.On("GetKnowledgeArticle", uint(7)).Return(want, nil)

	out, summary, err := knowledgeGet(ctx, mb, knowledgeGetInput{ID: 7})

	assert.NoError(t, err)
	assert.Equal(t, knowledgeArticleFrom(want), out)
	assert.Equal(t, uint(7), out.ID)
	assert.Equal(t, int64(12), out.ViewCount)
	assert.Contains(t, summary, "#7")
	mb.AssertExpectations(t)
}

func TestKnowledgeGetNotFound(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("knowledge:read"))

	mb.On("GetKnowledgeArticle", uint(99)).Return(nil, apperrors.NewNotFoundError("knowledge article not found"))

	out, _, err := knowledgeGet(ctx, mb, knowledgeGetInput{ID: 99})

	assert.Error(t, err)
	assert.Equal(t, knowledgeArticle{}, out)
	mb.AssertExpectations(t)
}

func TestKnowledgeList(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("knowledge:read"))

	pid := uint(5)
	in := knowledgeListInput{
		Page:      2,
		PageSize:  10,
		Status:    "published",
		Category:  "faq",
		ProductID: &pid,
		Search:    "vpn",
	}
	want := &knowledge.KnowledgeArticleListResponse{
		Data:       []knowledge.KnowledgeArticleResponse{{ID: 1, Title: "VPN setup", Status: "published"}},
		Total:      3,
		Page:       2,
		PageSize:   10,
		TotalPages: 1,
	}

	mb.On("ListKnowledgeArticles", 2, 10, mock.MatchedBy(func(f map[string]interface{}) bool {
		return f["status"] == "published" && f["category"] == "faq" && f["product_id"] == uint(5) && f["search"] == "vpn"
	})).Return(want, nil)

	out, summary, err := knowledgeList(ctx, mb, in)

	assert.NoError(t, err)
	assert.Equal(t, want.Total, out.Total)
	assert.Equal(t, want.Page, out.Page)
	assert.Equal(t, want.PageSize, out.PageSize)
	assert.Equal(t, want.TotalPages, out.TotalPages)
	assert.Len(t, out.Data, 1)
	assert.Equal(t, uint(1), out.Data[0].ID)
	assert.Equal(t, "VPN setup", out.Data[0].Title)
	assert.Contains(t, summary, "3 knowledge article(s)")
	mb.AssertExpectations(t)
}

func TestKnowledgeListDefaultsPaging(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("knowledge:read"))

	want := &knowledge.KnowledgeArticleListResponse{Total: 0, Page: 1, PageSize: 20, TotalPages: 0}
	mb.On("ListKnowledgeArticles", 1, 20, mock.MatchedBy(func(f map[string]interface{}) bool {
		return len(f) == 0
	})).Return(want, nil)

	out, _, err := knowledgeList(ctx, mb, knowledgeListInput{})

	assert.NoError(t, err)
	assert.Equal(t, int64(0), out.Total)
	assert.Equal(t, 1, out.Page)
	assert.Equal(t, 20, out.PageSize)
	assert.Empty(t, out.Data)
	mb.AssertExpectations(t)
}

func TestKnowledgeUpdate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("knowledge:write"))

	in := knowledgeUpdateInput{ID: 7, Title: "Updated Title", Status: "archived"}
	want := &knowledge.KnowledgeArticleResponse{ID: 7, Title: "Updated Title", Status: "archived", Version: 2}

	mb.On("UpdateKnowledgeArticle", uint(7), uint(1), mock.MatchedBy(func(req *knowledge.UpdateKnowledgeArticleRequest) bool {
		return req.Title == "Updated Title" && req.Status == "archived"
	})).Return(want, nil)

	out, summary, err := knowledgeUpdate(ctx, mb, in)

	assert.NoError(t, err)
	assert.Equal(t, knowledgeArticleFrom(want), out)
	assert.Equal(t, 2, out.Version)
	assert.Equal(t, "archived", out.Status)
	assert.Contains(t, summary, "version 2")
	mb.AssertExpectations(t)
}

func TestKnowledgeUpdateUnauthenticated(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(nil)

	out, _, err := knowledgeUpdate(ctx, mb, knowledgeUpdateInput{ID: 1})

	assert.ErrorIs(t, err, ErrUnauthenticated)
	assert.Equal(t, knowledgeArticle{}, out)
	mb.AssertExpectations(t)
}

func TestKnowledgeDelete(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("knowledge:write"))

	mb.On("DeleteKnowledgeArticle", uint(7), uint(1)).Return(nil)

	out, summary, err := knowledgeDelete(ctx, mb, knowledgeDeleteInput{ID: 7})

	assert.NoError(t, err)
	assert.Equal(t, knowledgeDeleteOutput{ID: 7, Deleted: true}, out)
	assert.Contains(t, summary, "#7")
	mb.AssertExpectations(t)
}

func TestKnowledgeDeleteError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("knowledge:write"))

	mb.On("DeleteKnowledgeArticle", uint(99), uint(1)).Return(apperrors.NewNotFoundError("knowledge article not found"))

	out, _, err := knowledgeDelete(ctx, mb, knowledgeDeleteInput{ID: 99})

	assert.Error(t, err)
	assert.False(t, out.Deleted)
	mb.AssertExpectations(t)
}

func TestKnowledgeDeleteUnauthenticated(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(nil)

	_, _, err := knowledgeDelete(ctx, mb, knowledgeDeleteInput{ID: 1})

	assert.ErrorIs(t, err, ErrUnauthenticated)
	mb.AssertExpectations(t)
}

func TestKnowledgeStats(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("knowledge:read"))

	want := &knowledge.KnowledgeArticleStatsResponse{
		TotalArticles:     10,
		PublishedArticles: 6,
		DraftArticles:     3,
		ArchivedArticles:  1,
		TotalViews:        250,
	}
	mb.On("GetKnowledgeArticleStats").Return(want, nil)

	out, summary, err := knowledgeStats(ctx, mb, knowledgeStatsInput{})

	assert.NoError(t, err)
	assert.Equal(t, knowledgeStatsOutputFrom(want), out)
	assert.Contains(t, summary, "10 article(s) total")
	mb.AssertExpectations(t)
}
