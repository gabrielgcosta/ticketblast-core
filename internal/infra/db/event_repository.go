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

func (r *PostgresEventRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if q, ok := ctx.Value(TxKey).(*sqlc.Queries); ok {
		return q
	}
	return r.queries
}

func (r *PostgresEventRepository) Create(ctx context.Context, event *entity.Event) (*entity.Event, error) {
	params := sqlc.CreateEventParams{
		Title:       event.Title,
		Description: pgtype.Text{String: event.Description, Valid: event.Description != ""},
		Location:    event.Location,
		EventDate:   pgtype.Timestamptz{Time: event.EventDate, Valid: true},
	}

	row, err := r.getQueries(ctx).CreateEvent(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create event in db: %w", err)
	}

	return &entity.Event{
		ID:          fromUUID(row.ID),
		Title:       row.Title,
		Description: row.Description.String,
		Location:    row.Location,
		EventDate:   row.EventDate.Time,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func (r *PostgresEventRepository) ListActive(ctx context.Context, now time.Time) ([]*entity.Event, error) {
	dbTime := pgtype.Timestamptz{Time: now, Valid: true}
	rows, err := r.getQueries(ctx).ListActiveEvents(ctx, dbTime)
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
