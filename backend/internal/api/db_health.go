package api

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type dbHealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// DBHealthHandler godoc
// @Summary      DB疎通確認
// @Description  PostgreSQLへの接続確認用エンドポイント
// @Tags         health
// @Produce      json
// @Success      200 {object} dbHealthResponse
// @Failure      503 {object} dbHealthResponse
// @Router       /health/db [get]
func DBHealthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(dbHealthResponse{Status: "error", Database: "disconnected"})
			return
		}

		json.NewEncoder(w).Encode(dbHealthResponse{Status: "ok", Database: "connected"})
	}
}
