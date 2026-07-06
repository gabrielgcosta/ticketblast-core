package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
	"github.com/stretchr/testify/assert"
)

type mockEventRepositoryForCreate struct {
	createFn func(ctx context.Context, event *entity.Event) (*entity.Event, error)
}

func (m *mockEventRepositoryForCreate) ListActive(ctx context.Context, now time.Time) ([]*entity.Event, error) {
	return nil, nil
}

func (m *mockEventRepositoryForCreate) Create(ctx context.Context, event *entity.Event) (*entity.Event, error) {
	return m.createFn(ctx, event)
}

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

type mockCacheServiceForCreate struct {
	setFn func(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

func (m *mockCacheServiceForCreate) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *mockCacheServiceForCreate) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return m.setFn(ctx, key, value, ttl)
}

func (m *mockCacheServiceForCreate) DecrBy(ctx context.Context, key string, decrement int64) (int64, error) {
	return 0, nil
}

func (m *mockCacheServiceForCreate) IncrBy(ctx context.Context, key string, increment int64) (int64, error) {
	return 0, nil
}

type mockTxManager struct{}

func (m *mockTxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func TestCreateEventUseCase_Success(t *testing.T) {
	mockEventRepo := &mockEventRepositoryForCreate{
		createFn: func(ctx context.Context, event *entity.Event) (*entity.Event, error) {
			return &entity.Event{
				ID:          "event-123",
				Title:       event.Title,
				Description: event.Description,
				Location:    event.Location,
				EventDate:   event.EventDate,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	mockTicketRepo := &mockTicketRepository{
		createFn: func(ctx context.Context, ticket *entity.Ticket) (*entity.Ticket, error) {
			return &entity.Ticket{
				ID:            "ticket-123",
				EventID:       ticket.EventID,
				Name:          ticket.Name,
				Price:         ticket.Price,
				TotalQuantity: ticket.TotalQuantity,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}, nil
		},
	}

	cacheSetCalled := false
	mockCache := &mockCacheServiceForCreate{
		setFn: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
			assert.Equal(t, "event:event-123:stock", key)
			assert.Equal(t, 100, value)
			assert.Equal(t, time.Duration(0), ttl)
			cacheSetCalled = true
			return nil
		},
	}

	uc := NewCreateEventUseCase(mockEventRepo, mockTicketRepo, mockCache, &mockTxManager{})
	input := CreateEventInput{
		Title:       "Big Match",
		Description: "Championship Final",
		Location:    "National Stadium",
		EventDate:   time.Now().Add(48 * time.Hour),
		TicketPrice: 75.50,
		Stock:       100,
	}

	output, err := uc.Execute(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "event-123", output.ID)
	assert.Equal(t, "ticket-123", output.TicketID)
	assert.Equal(t, 75.50, output.TicketPrice)
	assert.Equal(t, 100, output.Stock)
	assert.True(t, cacheSetCalled)
}

func TestCreateEventUseCase_EventRepoError(t *testing.T) {
	mockEventRepo := &mockEventRepositoryForCreate{
		createFn: func(ctx context.Context, event *entity.Event) (*entity.Event, error) {
			return nil, errors.New("db error")
		},
	}

	mockTicketRepo := &mockTicketRepository{
		createFn: func(ctx context.Context, ticket *entity.Ticket) (*entity.Ticket, error) {
			t.Fatal("Ticket creation should not be called")
			return nil, nil
		},
	}

	mockCache := &mockCacheServiceForCreate{
		setFn: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
			t.Fatal("Cache Set should not be called")
			return nil
		},
	}

	uc := NewCreateEventUseCase(mockEventRepo, mockTicketRepo, mockCache, &mockTxManager{})
	input := CreateEventInput{
		Title:       "Big Match",
		Location:    "Stadium",
		EventDate:   time.Now(),
		TicketPrice: 50.0,
		Stock:       50,
	}

	output, err := uc.Execute(context.Background(), input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create event")
	assert.Nil(t, output)
}
