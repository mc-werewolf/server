package api

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/mc-werewolf/server/backend/internal/addon"
	"github.com/mc-werewolf/server/backend/internal/github"
)

// NewRouter builds the /api router.
// Swagger UI is only mounted when devMode is true (dev.mc-werewolf.com only).
func NewRouter(devMode bool, pool *pgxpool.Pool, launcherConfig LauncherConfig) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", HealthHandler)
	mux.HandleFunc("GET /api/health/db", DBHealthHandler(pool))

	store := addon.NewStore(pool)
	ghClient := github.NewClient()

	// Admin (mutating) routes. Not gated at the Go level: perimeter auth is
	// handled by Caddy (basic_auth on the whole dev.mc-werewolf.com
	// subdomain, and on mc-werewolf.com/api/admin/* specifically).
	mux.HandleFunc("POST /api/admin/addons", RegisterAddonHandler(store, ghClient))
	mux.HandleFunc("POST /api/admin/addons/{id}/refresh", RefreshAddonHandler(store, ghClient))
	mux.HandleFunc("GET /api/admin/addons", ListAdminAddonsHandler(store))

	// Public routes, consumed by bds-launcher and the mc-werewolf.com site.
	mux.HandleFunc("GET /api/addons", ListAddonsHandler(store))
	mux.HandleFunc("GET /api/addons/{owner}/{repo}/versions", ListAddonVersionsHandler(store))
	mux.HandleFunc("GET /api/addons/{owner}/{repo}/versions/{tag}/download", DownloadAddonVersionHandler(store))
	mux.HandleFunc("GET /api/launcher/v1/config", LauncherConfigHandler(launcherConfig))

	if devMode {
		mux.Handle("/api/swagger/", httpSwagger.WrapHandler)
	}

	return mux
}
