package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates a PostgreSQL connection pool for the given DSN.
// It does not eagerly open a connection, so the server can start even if
// the database is briefly unavailable; callers should Ping when they need
// to verify connectivity.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, databaseURL)
}
