package db

import (
	"context"
	"fmt"

	"github.com/gabrielgcosta/ticketblast-core/db/sqlc"
	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresTicketRepository struct {
	queries *sqlc.Queries
}

func NewPostgresTicketRepository(queries *sqlc.Queries) *PostgresTicketRepository {
	return &PostgresTicketRepository{queries: queries}
}

func (r *PostgresTicketRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if q, ok := ctx.Value(TxKey).(*sqlc.Queries); ok {
		return q
	}
	return r.queries
}

func (r *PostgresTicketRepository) Create(ctx context.Context, ticket *entity.Ticket) (*entity.Ticket, error) {
	eventUID, err := toUUID(ticket.EventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event uuid: %w", err)
	}

	params := sqlc.CreateTicketParams{
		EventID:       eventUID,
		Name:          ticket.Name,
		Price:         toNumeric(ticket.Price),
		TotalQuantity: int32(ticket.TotalQuantity),
	}

	row, err := r.getQueries(ctx).CreateTicket(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create ticket in db: %w", err)
	}

	return &entity.Ticket{
		ID:            fromUUID(row.ID),
		EventID:       fromUUID(row.EventID),
		Name:          row.Name,
		Price:         fromNumeric(row.Price),
		TotalQuantity: int(row.TotalQuantity),
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}, nil
}

func (r *PostgresTicketRepository) GetByID(ctx context.Context, id string) (*entity.Ticket, error) {
	ticketUID, err := toUUID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ticket uuid: %w", err)
	}

	row, err := r.getQueries(ctx).GetTicketByID(ctx, ticketUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket by id: %w", err)
	}

	return &entity.Ticket{
		ID:            fromUUID(row.ID),
		EventID:       fromUUID(row.EventID),
		Name:          row.Name,
		Price:         fromNumeric(row.Price),
		TotalQuantity: int(row.TotalQuantity),
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}, nil
}

func (r *PostgresTicketRepository) UpdateStock(ctx context.Context, id string, quantity int) error {
	ticketUID, err := toUUID(id)
	if err != nil {
		return fmt.Errorf("invalid ticket uuid: %w", err)
	}

	params := sqlc.UpdateTicketStockParams{
		ID:            ticketUID,
		TotalQuantity: int32(quantity),
	}

	err = r.getQueries(ctx).UpdateTicketStock(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update ticket stock: %w", err)
	}

	return nil
}

// Helpers for numeric values
func toNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func fromNumeric(n pgtype.Numeric) float64 {
	var f float64
	if err := n.Scan(&f); err != nil {
		return 0.0
	}
	return f
}
