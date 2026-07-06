package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
	"github.com/gabrielgcosta/ticketblast-core/internal/infra/auth"
	"github.com/gabrielgcosta/ticketblast-core/internal/infra/web/middleware"
	"github.com/gabrielgcosta/ticketblast-core/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Define Mocks locally for Handler package tests
type mockTicketRepository struct {
	createFn      func(ctx context.Context, ticket *entity.Ticket) (*entity.Ticket, error)
	getByIDFn     func(ctx context.Context, id string) (*entity.Ticket, error)
	updateStockFn func(ctx context.Context, id string, quantity int) error
}

func (m *mockTicketRepository) Create(ctx context.Context, ticket *entity.Ticket) (*entity.Ticket, error) {
	return m.createFn(ctx, ticket)
}

func (m *mockTicketRepository) GetByID(ctx context.Context, id string) (*entity.Ticket, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockTicketRepository) UpdateStock(ctx context.Context, id string, quantity int) error {
	return m.updateStockFn(ctx, id, quantity)
}

type mockCacheServiceForPurchase struct {
	decrByFn func(ctx context.Context, key string, decrement int64) (int64, error)
	incrByFn func(ctx context.Context, key string, increment int64) (int64, error)
}

func (m *mockCacheServiceForPurchase) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *mockCacheServiceForPurchase) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func (m *mockCacheServiceForPurchase) DecrBy(ctx context.Context, key string, decrement int64) (int64, error) {
	return m.decrByFn(ctx, key, decrement)
}

func (m *mockCacheServiceForPurchase) IncrBy(ctx context.Context, key string, increment int64) (int64, error) {
	return m.incrByFn(ctx, key, increment)
}

type mockOrderRepository struct {
	createFn     func(ctx context.Context, order *entity.Order) (*entity.Order, error)
	createItemFn func(ctx context.Context, item *entity.OrderItem) (*entity.OrderItem, error)
}

func (m *mockOrderRepository) Create(ctx context.Context, order *entity.Order) (*entity.Order, error) {
	return m.createFn(ctx, order)
}

func (m *mockOrderRepository) CreateItem(ctx context.Context, item *entity.OrderItem) (*entity.OrderItem, error) {
	return m.createItemFn(ctx, item)
}

type mockTxManagerForHandler struct{}

func (m *mockTxManagerForHandler) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func TestPurchaseHandler_Purchase_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTicketRepo := &mockTicketRepository{
		getByIDFn: func(ctx context.Context, id string) (*entity.Ticket, error) {
			return &entity.Ticket{
				ID:            "ticket-123",
				EventID:       "event-123",
				Name:          "VIP",
				Price:         100.00,
				TotalQuantity: 10,
			}, nil
		},
		updateStockFn: func(ctx context.Context, id string, quantity int) error {
			return nil
		},
	}

	mockOrderRepo := &mockOrderRepository{
		createFn: func(ctx context.Context, order *entity.Order) (*entity.Order, error) {
			return &entity.Order{
				ID:          "order-123",
				UserID:      order.UserID,
				Status:      order.Status,
				TotalAmount: order.TotalAmount,
			}, nil
		},
		createItemFn: func(ctx context.Context, item *entity.OrderItem) (*entity.OrderItem, error) {
			return item, nil
		},
	}

	mockCache := &mockCacheServiceForPurchase{
		decrByFn: func(ctx context.Context, key string, decrement int64) (int64, error) {
			return 8, nil
		},
	}

	purchaseUC := usecase.NewPurchaseUseCase(mockTicketRepo, mockOrderRepo, mockCache, &mockTxManagerForHandler{})
	handler := NewPurchaseHandler(purchaseUC)

	tokenEngine := auth.NewTokenEngine("secret")
	r := gin.New()
	r.Use(middleware.Auth(tokenEngine))
	r.POST("/orders", handler.Purchase)

	// Generate valid token
	token, _ := tokenEngine.GenerateToken("user-123", "customer")

	body := []byte(`{"event_id":"event-123","ticket_id":"ticket-123","quantity":2}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"order_id":"order-123"`)
}

func TestPurchaseHandler_Purchase_SoldOut(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockCache := &mockCacheServiceForPurchase{
		decrByFn: func(ctx context.Context, key string, decrement int64) (int64, error) {
			return -1, nil
		},
		incrByFn: func(ctx context.Context, key string, increment int64) (int64, error) {
			return 0, nil
		},
	}

	purchaseUC := usecase.NewPurchaseUseCase(nil, nil, mockCache, &mockTxManagerForHandler{})
	handler := NewPurchaseHandler(purchaseUC)

	tokenEngine := auth.NewTokenEngine("secret")
	r := gin.New()
	r.Use(middleware.Auth(tokenEngine))
	r.POST("/orders", handler.Purchase)

	token, _ := tokenEngine.GenerateToken("user-123", "customer")

	body := []byte(`{"event_id":"event-123","ticket_id":"ticket-123","quantity":5}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Tickets sold out")
}
