package db

import (
	"context"
	"fmt"
	"time"

	"github.com/gabrielgcosta/ticketblast-core/db/sqlc"
	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresEventRepository struct {
	queries *sqlc.Queries
}

func NewPostgresEventRepository(queries *sqlc.Queries) *PostgresEventRepository {
	return &PostgresEventRepository{queries: queries}
}

func (r *PostgresEventRepository) ListActive(ctx context.Context, now time.Time) ([]*entity.Event, error) {
	dbTime := pgtype.Timestamptz{Time: now, Valid: true}
	rows, err := r.queries.ListActiveEvents(ctx, dbTime)
	if err != nil {
		return nil, fmt.Errorf("failed to list active events from db: %w", err)
	}

	events := make([]*entity.Event, len(rows))
	for i, row := range rows {
		events[i] = &entity.Event{
			ID:          fromUUID(row.ID),
			Title:       row.Title,
			Description: row.Description.String,
			Location:    row.Location,
			EventDate:   row.EventDate.Time,
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
		}
	}

	return events, nil
}
