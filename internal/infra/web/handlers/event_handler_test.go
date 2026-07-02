package handlers

import (
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
}

func (m *mockEventRepository) ListActive(ctx context.Context, now time.Time) ([]*entity.Event, error) {
	return m.listActiveFn(ctx, now)
}

type mockCacheService struct {
	getFn func(ctx context.Context, key string) (string, error)
	setFn func(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

func (m *mockCacheService) Get(ctx context.Context, key string) (string, error) {
	return m.getFn(ctx, key)
}

func (m *mockCacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return m.setFn(ctx, key, value, ttl)
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
	handler := NewEventHandler(listActiveUC)

	r := gin.New()
	r.GET("/events/active", handler.ListActive)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/events/active", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"title":"Awesome Concert"`)
}
