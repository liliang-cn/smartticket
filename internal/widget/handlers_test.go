package widget

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/ticket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// handlerTestDB sets up an in-memory SQLite database for handler tests.
func handlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Customer{},
		&models.User{},
		&models.Ticket{},
		&models.Message{},
		&models.SLATemplate{},
		&models.SLARule{},
	))
	return db
}

// fakeHandlerTicketService is a ticket service fake for handler tests.
type fakeHandlerTicketService struct {
	db      *gorm.DB
	counter uint
}

func (f *fakeHandlerTicketService) CreateTicket(_ authz.Actor, _ uint, req *ticket.CreateTicketRequest) (*ticket.TicketResponse, error) {
	f.counter++
	tkt := &models.Ticket{
		TicketNumber:   fmt.Sprintf("TK-H%d", f.counter),
		Title:          req.Title,
		Description:    req.Description,
		Status:         "open",
		Priority:       req.Priority,
		Severity:       req.Severity,
		RequesterName:  req.RequesterName,
		RequesterEmail: req.RequesterEmail,
	}
	if req.CustomerID != nil {
		tkt.CustomerID = req.CustomerID
	}
	if err := f.db.Create(tkt).Error; err != nil {
		return nil, err
	}
	return &ticket.TicketResponse{
		ID:             tkt.ID,
		TicketNumber:   tkt.TicketNumber,
		Title:          tkt.Title,
		RequesterName:  tkt.RequesterName,
		RequesterEmail: tkt.RequesterEmail,
	}, nil
}

func (f *fakeHandlerTicketService) CreateMessage(_ authz.Actor, ticketID, _ uint, req *ticket.CreateMessageRequest) (*ticket.MessageResponse, error) {
	msg := &models.Message{
		TicketID:    ticketID,
		Content:     req.Content,
		ContentType: req.ContentType,
	}
	if err := f.db.Create(msg).Error; err != nil {
		return nil, err
	}
	return &ticket.MessageResponse{
		ID:      msg.ID,
		TicketID: ticketID,
		Content: req.Content,
	}, nil
}

func (f *fakeHandlerTicketService) ListMessages(_ authz.Actor, ticketID uint) ([]ticket.MessageResponse, error) {
	var msgs []models.Message
	if err := f.db.Where("ticket_id = ? AND is_internal = ?", ticketID, false).
		Order("created_at ASC").Find(&msgs).Error; err != nil {
		return nil, err
	}
	out := make([]ticket.MessageResponse, len(msgs))
	for i, m := range msgs {
		out[i] = ticket.MessageResponse{ID: m.ID, TicketID: m.TicketID, Content: m.Content}
	}
	return out, nil
}

func newHandlerRouter(t *testing.T) (*gin.Engine, *Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := handlerTestDB(t)
	fake := &fakeHandlerTicketService{db: db}
	svc := NewService(db, fake, testSecret)
	h := NewHandlers(svc)

	r := gin.New()
	r.POST("/widget/session", h.StartSession)
	r.POST("/widget/messages", h.PostMessage)
	r.GET("/widget/messages", h.History)
	return r, svc
}

// TestHandler_StartSession_Returns200WithToken verifies the happy path for
// POST /widget/session.
func TestHandler_StartSession_Returns200WithToken(t *testing.T) {
	r, _ := newHandlerRouter(t)

	payload := `{"email":"handler@test.com","name":"Handler Tester","message":"hello handler"}`
	req := httptest.NewRequest(http.MethodPost, "/widget/session", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool            `json:"success"`
		Data    SessionResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.NotEmpty(t, resp.Data.Token)
	require.NotZero(t, resp.Data.TicketID)

	// Token must be parseable.
	tid, err := ParseToken(resp.Data.Token, testSecret)
	require.NoError(t, err)
	require.Equal(t, resp.Data.TicketID, tid)
}

// TestHandler_PostMessage_Returns200(t *testing.T) verifies POST /widget/messages.
func TestHandler_PostMessage_Returns200(t *testing.T) {
	r, _ := newHandlerRouter(t)

	// First create a session.
	payload := `{"email":"poster@test.com","message":"initial"}`
	req := httptest.NewRequest(http.MethodPost, "/widget/session", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var sessResp struct {
		Data SessionResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sessResp))

	// Now post a message.
	msgBody := `{"message":"follow up"}`
	req2 := httptest.NewRequest(http.MethodPost, "/widget/messages", bytes.NewBufferString(msgBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+sessResp.Data.Token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusOK, w2.Code)
	var msgResp struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &msgResp))
	require.True(t, msgResp.Success)
}

// TestHandler_History_Returns200(t *testing.T) verifies GET /widget/messages.
func TestHandler_History_Returns200(t *testing.T) {
	r, _ := newHandlerRouter(t)

	// Create a session.
	payload := `{"email":"history@test.com","message":"my first message"}`
	req := httptest.NewRequest(http.MethodPost, "/widget/session", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var sessResp struct {
		Data SessionResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sessResp))

	// Get history via query param.
	req2 := httptest.NewRequest(http.MethodGet, "/widget/messages?token="+sessResp.Data.Token, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusOK, w2.Code)
}

// TestHandler_StartSession_AnonymousRequest verifies anonymous sessions work.
func TestHandler_StartSession_AnonymousRequest(t *testing.T) {
	r, _ := newHandlerRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/widget/session", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Success bool            `json:"success"`
		Data    SessionResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.NotEmpty(t, resp.Data.Token)
}
