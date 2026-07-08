package api

import (
	"encoding/json"
	"net/http"
)

type healthResponse struct {
	Status string `json:"status"`
}

// HealthHandler godoc
// @Summary      ヘルスチェック
// @Description  APIサーバーの生存確認用エンドポイント
// @Tags         health
// @Produce      json
// @Success      200 {object} healthResponse
// @Router       /health [get]
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
}
