package usecase

import (
	"context"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
)

type TicketRepository interface {
	Create(ctx context.Context, ticket *entity.Ticket) (*entity.Ticket, error)
	GetByID(ctx context.Context, id string) (*entity.Ticket, error)
	UpdateStock(ctx context.Context, id string, quantity int) error
}
