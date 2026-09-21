package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the shared connection pool. One pool for the whole backend —
// no need for anything fancier at this scale.
var Pool *pgxpool.Pool

func Connect(ctx context.Context) error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://gpumarket:gpumarket@localhost:5432/gpumarket?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return err
	}
	Pool = pool
	return nil
}
