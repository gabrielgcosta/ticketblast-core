package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
	"github.com/gabrielgcosta/ticketblast-core/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockEventRepository struct {
	listActiveFn func(ctx context.Context, now time.Time) ([]*entity.Event, error)
	createFn     func(ctx context.Context, event *entity.Event) (*entity.Event, error)
}

func (m *mockEventRepository) ListActive(ctx context.Context, now time.Time) ([]*entity.Event, error) {
	return m.listActiveFn(ctx, now)
}

func (m *mockEventRepository) Create(ctx context.Context, event *entity.Event) (*entity.Event, error) {
	if m.createFn != nil {
		return m.createFn(ctx, event)
	}
	return nil, nil
}

type mockTicketRepositoryForCreate struct {
	createFn func(ctx context.Context, ticket *entity.Ticket) (*entity.Ticket, error)
}

func (m *mockTicketRepositoryForCreate) Create(ctx context.Context, ticket *entity.Ticket) (*entity.Ticket, error) {
	return m.createFn(ctx, ticket)
}

func (m *mockTicketRepositoryForCreate) GetByID(ctx context.Context, id string) (*entity.Ticket, error) {
	return nil, nil
}

func (m *mockTicketRepositoryForCreate) UpdateStock(ctx context.Context, id string, quantity int) error {
	return nil
}

type mockCacheService struct {
	getFn func(ctx context.Context, key string) (string, error)
	setFn func(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

func (m *mockCacheService) Get(ctx context.Context, key string) (string, error) {
	if m.getFn != nil {
		return m.getFn(ctx, key)
	}
	return "", nil
}

func (m *mockCacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m.setFn != nil {
		return m.setFn(ctx, key, value, ttl)
	}
	return nil
}

func (m *mockCacheService) DecrBy(ctx context.Context, key string, decrement int64) (int64, error) {
	return 0, nil
}

func (m *mockCacheService) IncrBy(ctx context.Context, key string, increment int64) (int64, error) {
	return 0, nil
}

type mockTxManager struct{}

func (m *mockTxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func TestEventHandler_ListActive_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &mockEventRepository{
		listActiveFn: func(ctx context.Context, now time.Time) ([]*entity.Event, error) {
			return []*entity.Event{
				{
					ID:        "event-1",
					Title:     "Awesome Concert",
					Location:  "Arena",
					EventDate: time.Now().Add(24 * time.Hour),
				},
			}, nil
		},
	}

	mockCache := &mockCacheService{
		getFn: func(ctx context.Context, key string) (string, error) {
			return "", nil // cache miss
		},
		setFn: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
			return nil
		},
	}

	listActiveUC := usecase.NewListActiveEventsUseCase(mockRepo, mockCache)
	handler := NewEventHandler(listActiveUC, nil)

	r := gin.New()
	r.GET("/events/active", handler.ListActive)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/events/active", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"title":"Awesome Concert"`)
}

func TestEventHandler_CreateEvent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockEventRepo := &mockEventRepository{
		createFn: func(ctx context.Context, event *entity.Event) (*entity.Event, error) {
			return &entity.Event{
				ID:        "event-123",
				Title:     event.Title,
				Location:  event.Location,
				EventDate: event.EventDate,
			}, nil
		},
	}

	mockTicketRepo := &mockTicketRepositoryForCreate{
		createFn: func(ctx context.Context, ticket *entity.Ticket) (*entity.Ticket, error) {
			return &entity.Ticket{
				ID:            "ticket-123",
				EventID:       ticket.EventID,
				Name:          ticket.Name,
				Price:         ticket.Price,
				TotalQuantity: ticket.TotalQuantity,
			}, nil
		},
	}

	mockCache := &mockCacheService{
		setFn: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
			return nil
		},
	}

	createEventUC := usecase.NewCreateEventUseCase(mockEventRepo, mockTicketRepo, mockCache, &mockTxManager{})
	handler := NewEventHandler(nil, createEventUC)

	r := gin.New()
	r.POST("/events", handler.CreateEvent)

	body := []byte(`{"title":"Rock Show","description":"Concert","location":"Arena","event_date":"2026-07-02T10:00:00Z","ticket_price":120.00,"stock":500}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/events", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"event-123"`)
	assert.Contains(t, w.Body.String(), `"ticket_id":"ticket-123"`)
}
