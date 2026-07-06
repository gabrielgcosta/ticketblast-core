package usecase

import (
	"context"
	"time"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
)

type EventRepository interface {
	ListActive(ctx context.Context, now time.Time) ([]*entity.Event, error)
	Create(ctx context.Context, event *entity.Event) (*entity.Event, error)
}
