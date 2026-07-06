package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
	"github.com/stretchr/testify/assert"
)

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

func TestPurchaseUseCase_Success(t *testing.T) {
	redisKey := "event:event-123:stock"
	decrCalled := false
	incrCalled := false

	mockCache := &mockCacheServiceForPurchase{
		decrByFn: func(ctx context.Context, key string, decrement int64) (int64, error) {
			assert.Equal(t, redisKey, key)
			assert.Equal(t, int64(2), decrement)
			decrCalled = true
			return 8, nil // Remaining stock: 8
		},
		incrByFn: func(ctx context.Context, key string, increment int64) (int64, error) {
			t.Fatal("IncrBy should not be called on success")
			return 0, nil
		},
	}

	mockTicketRepo := &mockTicketRepository{
		getByIDFn: func(ctx context.Context, id string) (*entity.Ticket, error) {
			assert.Equal(t, "ticket-123", id)
			return &entity.Ticket{
				ID:            "ticket-123",
				EventID:       "event-123",
				Name:          "VIP",
				Price:         150.00,
				TotalQuantity: 10,
			}, nil
		},
		updateStockFn: func(ctx context.Context, id string, quantity int) error {
			assert.Equal(t, "ticket-123", id)
			assert.Equal(t, 2, quantity)
			return nil
		},
	}

	mockOrderRepo := &mockOrderRepository{
		createFn: func(ctx context.Context, order *entity.Order) (*entity.Order, error) {
			assert.Equal(t, "user-123", order.UserID)
			assert.Equal(t, entity.OrderStatusCompleted, order.Status)
			assert.Equal(t, 300.00, order.TotalAmount)
			return &entity.Order{
				ID:          "order-123",
				UserID:      order.UserID,
				Status:      order.Status,
				TotalAmount: order.TotalAmount,
				CreatedAt:   time.Now(),
			}, nil
		},
		createItemFn: func(ctx context.Context, item *entity.OrderItem) (*entity.OrderItem, error) {
			assert.Equal(t, "order-123", item.OrderID)
			assert.Equal(t, "ticket-123", item.TicketID)
			assert.Equal(t, 2, item.Quantity)
			assert.Equal(t, 150.00, item.UnitPrice)
			return item, nil
		},
	}

	uc := NewPurchaseUseCase(mockTicketRepo, mockOrderRepo, mockCache, &mockTxManager{})
	input := PurchaseInput{
		UserID:   "user-123",
		EventID:  "event-123",
		TicketID: "ticket-123",
		Quantity: 2,
	}

	output, err := uc.Execute(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "order-123", output.OrderID)
	assert.Equal(t, 300.00, output.TotalAmount)
	assert.Equal(t, string(entity.OrderStatusCompleted), output.Status)
	assert.True(t, decrCalled)
	assert.False(t, incrCalled)
}

func TestPurchaseUseCase_SoldOut(t *testing.T) {
	redisKey := "event:event-123:stock"
	decrCalled := false
	incrCalled := false

	mockCache := &mockCacheServiceForPurchase{
		decrByFn: func(ctx context.Context, key string, decrement int64) (int64, error) {
			assert.Equal(t, redisKey, key)
			assert.Equal(t, int64(5), decrement)
			decrCalled = true
			return -3, nil // Insufficient stock (e.g. only 2 available, tried to buy 5)
		},
		incrByFn: func(ctx context.Context, key string, increment int64) (int64, error) {
			assert.Equal(t, redisKey, key)
			assert.Equal(t, int64(5), increment)
			incrCalled = true
			return 2, nil
		},
	}

	mockTicketRepo := &mockTicketRepository{
		getByIDFn: func(ctx context.Context, id string) (*entity.Ticket, error) {
			t.Fatal("Ticket query should not occur when sold out in cache")
			return nil, nil
		},
	}

	mockOrderRepo := &mockOrderRepository{}

	uc := NewPurchaseUseCase(mockTicketRepo, mockOrderRepo, mockCache, &mockTxManager{})
	input := PurchaseInput{
		UserID:   "user-123",
		EventID:  "event-123",
		TicketID: "ticket-123",
		Quantity: 5,
	}

	output, err := uc.Execute(context.Background(), input)

	assert.ErrorIs(t, err, ErrSoldOut)
	assert.Nil(t, output)
	assert.True(t, decrCalled)
	assert.True(t, incrCalled)
}

func TestPurchaseUseCase_DBFailure_RevertsRedis(t *testing.T) {
	redisKey := "event:event-123:stock"
	decrCalled := false
	incrCalled := false

	mockCache := &mockCacheServiceForPurchase{
		decrByFn: func(ctx context.Context, key string, decrement int64) (int64, error) {
			assert.Equal(t, redisKey, key)
			decrCalled = true
			return 8, nil
		},
		incrByFn: func(ctx context.Context, key string, increment int64) (int64, error) {
			assert.Equal(t, redisKey, key)
			assert.Equal(t, int64(2), increment)
			incrCalled = true
			return 10, nil
		},
	}

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
			return errors.New("database down")
		},
	}

	mockOrderRepo := &mockOrderRepository{
		createFn: func(ctx context.Context, order *entity.Order) (*entity.Order, error) {
			return &entity.Order{
				ID: "order-123",
			}, nil
		},
		createItemFn: func(ctx context.Context, item *entity.OrderItem) (*entity.OrderItem, error) {
			return item, nil
		},
	}

	uc := NewPurchaseUseCase(mockTicketRepo, mockOrderRepo, mockCache, &mockTxManager{})
	input := PurchaseInput{
		UserID:   "user-123",
		EventID:  "event-123",
		TicketID: "ticket-123",
		Quantity: 2,
	}

	output, err := uc.Execute(context.Background(), input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database down")
	assert.Nil(t, output)
	assert.True(t, decrCalled)
	assert.True(t, incrCalled)
}
