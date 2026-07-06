package db

import (
	"context"
	"fmt"

	"github.com/gabrielgcosta/ticketblast-core/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKeyType struct{}

var TxKey = txKeyType{}

type PostgresTxManager struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPostgresTxManager(pool *pgxpool.Pool, queries *sqlc.Queries) *PostgresTxManager {
	return &PostgresTxManager{
		pool:    pool,
		queries: queries,
	}
}

func (m *PostgresTxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := m.queries.WithTx(tx)
	ctxWithTx := context.WithValue(ctx, TxKey, qtx)

	if err := fn(ctxWithTx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
