package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrSoldOut             = errors.New("tickets sold out")
	ErrTicketNotFound      = errors.New("ticket not found")
	ErrTicketEventMismatch = errors.New("ticket does not belong to the specified event")
	ErrInvalidQuantity     = errors.New("invalid purchase quantity")
)

type PurchaseInput struct {
	UserID   string `json:"user_id"`
	EventID  string `json:"event_id"`
	TicketID string `json:"ticket_id"`
	Quantity int    `json:"quantity"`
}

type PurchaseOutput struct {
	OrderID     string  `json:"order_id"`
	TotalAmount float64 `json:"total_amount"`
	Status      string  `json:"status"`
}

type PurchaseUseCase struct {
	ticketRepo TicketRepository
	cache      CacheService
	publisher  EventPublisher
}

func NewPurchaseUseCase(
	ticketRepo TicketRepository,
	cache CacheService,
	publisher EventPublisher,
) *PurchaseUseCase {
	return &PurchaseUseCase{
		ticketRepo: ticketRepo,
		cache:      cache,
		publisher:  publisher,
	}
}

func (uc *PurchaseUseCase) Execute(ctx context.Context, input PurchaseInput) (*PurchaseOutput, error) {
	if input.Quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	redisKey := fmt.Sprintf("event:%s:stock", input.EventID)

	// 1. Atomic Redis Decrement (Before any database operations or long validations)
	newStock, err := uc.cache.DecrBy(ctx, redisKey, int64(input.Quantity))
	if err != nil {
		return nil, fmt.Errorf("failed to decrement inventory in cache: %w", err)
	}

	// 2. If newStock < 0, inventory is exhausted. Revert immediately and return error.
	if newStock < 0 {
		_, revertErr := uc.cache.IncrBy(ctx, redisKey, int64(input.Quantity))
		if revertErr != nil {
			return nil, fmt.Errorf("revert failed after stock exhaustion: %w, original error: %w", revertErr, ErrSoldOut)
		}
		return nil, ErrSoldOut
	}

	// 3. Fetch ticket details to validate and compute amount
	ticket, err := uc.ticketRepo.GetByID(ctx, input.TicketID)
	if err != nil {
		_, revertErr := uc.cache.IncrBy(ctx, redisKey, int64(input.Quantity))
		if revertErr != nil {
			return nil, fmt.Errorf("revert failed after ticket query failure: %w, original error: %w", revertErr, ErrTicketNotFound)
		}
		return nil, ErrTicketNotFound
	}

	// 4. Verify ticket belongs to the event
	if ticket.EventID != input.EventID {
		_, revertErr := uc.cache.IncrBy(ctx, redisKey, int64(input.Quantity))
		if revertErr != nil {
			return nil, fmt.Errorf("revert failed after ticket event mismatch: %w, original error: %w", revertErr, ErrTicketEventMismatch)
		}
		return nil, ErrTicketEventMismatch
	}

	// 5. Generate unique Order ID
	orderID := uuid.New().String()
	totalAmount := ticket.Price * float64(input.Quantity)

	// 6. Build and publish event to RabbitMQ
	event := &OrderCreatedEvent{
		OrderID:  orderID,
		UserID:   input.UserID,
		EventID:  input.EventID,
		TicketID: input.TicketID,
		Quantity: input.Quantity,
		Price:    ticket.Price,
	}

	if err := uc.publisher.PublishOrderCreated(ctx, event); err != nil {
		_, revertErr := uc.cache.IncrBy(ctx, redisKey, int64(input.Quantity))
		if revertErr != nil {
			return nil, fmt.Errorf("revert failed after publisher error: %w, publisher error: %w", revertErr, err)
		}
		return nil, fmt.Errorf("failed to publish order event: %w", err)
	}

	return &PurchaseOutput{
		OrderID:     orderID,
		TotalAmount: totalAmount,
		Status:      "pending",
	}, nil
}
