package api

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouter builds the /api router.
// Swagger UI is only mounted when devMode is true (dev.mc-werewolf.com only).
func NewRouter(devMode bool, pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", HealthHandler)
	mux.HandleFunc("GET /api/health/db", DBHealthHandler(pool))

	if devMode {
		mux.Handle("/api/swagger/", httpSwagger.WrapHandler)
	}

	return mux
}
