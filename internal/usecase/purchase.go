package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
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
	orderRepo  OrderRepository
	cache      CacheService
	txManager  TxManager
}

func NewPurchaseUseCase(
	ticketRepo TicketRepository,
	orderRepo OrderRepository,
	cache CacheService,
	txManager TxManager,
) *PurchaseUseCase {
	return &PurchaseUseCase{
		ticketRepo: ticketRepo,
		orderRepo:  orderRepo,
		cache:      cache,
		txManager:  txManager,
	}
}

func (uc *PurchaseUseCase) Execute(ctx context.Context, input PurchaseInput) (*PurchaseOutput, error) {
	if input.Quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	redisKey := fmt.Sprintf("event:%s:stock", input.EventID)

	// 1. Atomic Redis Decrement
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

	// 3. Database transaction for order placement
	var createdOrder *entity.Order
	dbErr := uc.txManager.RunInTx(ctx, func(ctx context.Context) error {
		// A. Fetch ticket details
		ticket, err := uc.ticketRepo.GetByID(ctx, input.TicketID)
		if err != nil {
			return ErrTicketNotFound
		}

		// B. Verify ticket belongs to event
		if ticket.EventID != input.EventID {
			return ErrTicketEventMismatch
		}

		// C. Double-check stock consistency in DB
		if ticket.TotalQuantity < input.Quantity {
			return errors.New("insufficient inventory in database")
		}

		// D. Calculate total amount
		totalAmount := ticket.Price * float64(input.Quantity)

		// E. Create order
		order := &entity.Order{
			UserID:      input.UserID,
			Status:      entity.OrderStatusCompleted,
			TotalAmount: totalAmount,
		}
		createdOrder, err = uc.orderRepo.Create(ctx, order)
		if err != nil {
			return fmt.Errorf("failed to save order: %w", err)
		}

		// F. Create order item
		item := &entity.OrderItem{
			OrderID:   createdOrder.ID,
			TicketID:  input.TicketID,
			Quantity:  input.Quantity,
			UnitPrice: ticket.Price,
		}
		_, err = uc.orderRepo.CreateItem(ctx, item)
		if err != nil {
			return fmt.Errorf("failed to save order item: %w", err)
		}

		// G. Reduce ticket stock in database
		err = uc.ticketRepo.UpdateStock(ctx, input.TicketID, input.Quantity)
		if err != nil {
			return fmt.Errorf("failed to update ticket stock in database: %w", err)
		}

		return nil
	})

	// 4. If database transaction fails, revert Redis decrement
	if dbErr != nil {
		_, revertErr := uc.cache.IncrBy(ctx, redisKey, int64(input.Quantity))
		if revertErr != nil {
			return nil, fmt.Errorf("revert failed after database error: %w, database error: %w", revertErr, dbErr)
		}
		return nil, dbErr
	}

	return &PurchaseOutput{
		OrderID:     createdOrder.ID,
		TotalAmount: createdOrder.TotalAmount,
		Status:      string(createdOrder.Status),
	}, nil
}
