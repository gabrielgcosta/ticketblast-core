package usecase

import (
	"context"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
)

type OrderRepository interface {
	Create(ctx context.Context, order *entity.Order) (*entity.Order, error)
	CreateItem(ctx context.Context, item *entity.OrderItem) (*entity.OrderItem, error)
}
