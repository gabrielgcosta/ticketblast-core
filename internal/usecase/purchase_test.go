package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

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

type mockEventPublisher struct {
	publishFn func(ctx context.Context, event *OrderCreatedEvent) error
}

func (m *mockEventPublisher) PublishOrderCreated(ctx context.Context, event *OrderCreatedEvent) error {
	return m.publishFn(ctx, event)
}

func TestPurchaseUseCase_Success(t *testing.T) {
	redisKey := "event:event-123:stock"
	decrCalled := false
	incrCalled := false
	publishCalled := false

	mockCache := &mockCacheServiceForPurchase{
		decrByFn: func(ctx context.Context, key string, decrement int64) (int64, error) {
			assert.Equal(t, redisKey, key)
			assert.Equal(t, int64(2), decrement)
			decrCalled = true
			return 8, nil
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
	}

	mockPublisher := &mockEventPublisher{
		publishFn: func(ctx context.Context, event *OrderCreatedEvent) error {
			assert.NotEmpty(t, event.OrderID)
			assert.NoError(t, uuid.Validate(event.OrderID))
			assert.Equal(t, "user-123", event.UserID)
			assert.Equal(t, "event-123", event.EventID)
			assert.Equal(t, "ticket-123", event.TicketID)
			assert.Equal(t, 2, event.Quantity)
			assert.Equal(t, 150.00, event.Price)
			publishCalled = true
			return nil
		},
	}

	uc := NewPurchaseUseCase(mockTicketRepo, mockCache, mockPublisher)
	input := PurchaseInput{
		UserID:   "user-123",
		EventID:  "event-123",
		TicketID: "ticket-123",
		Quantity: 2,
	}

	output, err := uc.Execute(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output.OrderID)
	assert.Equal(t, 300.00, output.TotalAmount)
	assert.Equal(t, "pending", output.Status)
	assert.True(t, decrCalled)
	assert.False(t, incrCalled)
	assert.True(t, publishCalled)
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
			return -3, nil
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

	mockPublisher := &mockEventPublisher{
		publishFn: func(ctx context.Context, event *OrderCreatedEvent) error {
			t.Fatal("Publish should not be called when sold out")
			return nil
		},
	}

	uc := NewPurchaseUseCase(mockTicketRepo, mockCache, mockPublisher)
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

func TestPurchaseUseCase_PublishFailure_RevertsRedis(t *testing.T) {
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
	}

	mockPublisher := &mockEventPublisher{
		publishFn: func(ctx context.Context, event *OrderCreatedEvent) error {
			return errors.New("rabbitmq down")
		},
	}

	uc := NewPurchaseUseCase(mockTicketRepo, mockCache, mockPublisher)
	input := PurchaseInput{
		UserID:   "user-123",
		EventID:  "event-123",
		TicketID: "ticket-123",
		Quantity: 2,
	}

	output, err := uc.Execute(context.Background(), input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rabbitmq down")
	assert.Nil(t, output)
	assert.True(t, decrCalled)
	assert.True(t, incrCalled)
}
