package db

import (
	"context"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BuildURL constructs a postgres:// connection string, percent-encoding the
// user and password so that special characters (e.g. from a generated
// secret) can't produce an invalid or misparsed URL.
func BuildURL(host, port, user, password, dbname string) string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   host + ":" + port,
		Path:   "/" + dbname,
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	return u.String()
}

// Connect creates a PostgreSQL connection pool for the given DSN.
// It does not eagerly open a connection, so the server can start even if
// the database is briefly unavailable; callers should Ping when they need
// to verify connectivity.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, databaseURL)
}
