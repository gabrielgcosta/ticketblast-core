package db

import (
	"context"
	"fmt"

	"github.com/gabrielgcosta/ticketblast-core/db/sqlc"
	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
)

type PostgresOrderRepository struct {
	queries *sqlc.Queries
}

func NewPostgresOrderRepository(queries *sqlc.Queries) *PostgresOrderRepository {
	return &PostgresOrderRepository{queries: queries}
}

func (r *PostgresOrderRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if q, ok := ctx.Value(TxKey).(*sqlc.Queries); ok {
		return q
	}
	return r.queries
}

func (r *PostgresOrderRepository) Create(ctx context.Context, order *entity.Order) (*entity.Order, error) {
	userUID, err := toUUID(order.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user uuid: %w", err)
	}

	params := sqlc.CreateOrderParams{
		UserID:      userUID,
		Status:      string(order.Status),
		TotalAmount: toNumeric(order.TotalAmount),
	}

	row, err := r.getQueries(ctx).CreateOrder(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create order in db: %w", err)
	}

	return &entity.Order{
		ID:          fromUUID(row.ID),
		UserID:      fromUUID(row.UserID),
		Status:      entity.OrderStatus(row.Status),
		TotalAmount: fromNumeric(row.TotalAmount),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func (r *PostgresOrderRepository) CreateItem(ctx context.Context, item *entity.OrderItem) (*entity.OrderItem, error) {
	orderUID, err := toUUID(item.OrderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order uuid: %w", err)
	}

	ticketUID, err := toUUID(item.TicketID)
	if err != nil {
		return nil, fmt.Errorf("invalid ticket uuid: %w", err)
	}

	params := sqlc.CreateOrderItemParams{
		OrderID:   orderUID,
		TicketID:  ticketUID,
		Quantity:  int32(item.Quantity),
		UnitPrice: toNumeric(item.UnitPrice),
	}

	row, err := r.getQueries(ctx).CreateOrderItem(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create order item in db: %w", err)
	}

	return &entity.OrderItem{
		ID:        fromUUID(row.ID),
		OrderID:   fromUUID(row.OrderID),
		TicketID:  fromUUID(row.TicketID),
		Quantity:  int(row.Quantity),
		UnitPrice: fromNumeric(row.UnitPrice),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}
