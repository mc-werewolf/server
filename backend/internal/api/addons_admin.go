package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mc-werewolf/server/backend/internal/addon"
	"github.com/mc-werewolf/server/backend/internal/github"
)

type registerAddonRequest struct {
	URL string `json:"url"`
}

// RegisterAddonHandler godoc
// @Summary      アドオンの登録・同期
// @Description  GitHubリポジトリURLを受け取り、owner/repoを抽出してReleasesを取得し、各バージョンのBPパックからproperties.jsを抽出して保存する
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        body body registerAddonRequest true "登録するリポジトリのURL"
// @Success      200 {object} addon.SyncResult
// @Failure      400 {object} errorResponse
// @Failure      502 {object} errorResponse
// @Router       /admin/addons [post]
func RegisterAddonHandler(store *addon.Store, gh *github.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerAddonRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		owner, repo, err := github.ParseRepoURL(req.URL)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		result, err := addon.SyncAddon(r.Context(), gh, store, owner, repo)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

// RefreshAddonHandler godoc
// @Summary      アドオンの再同期
// @Description  既に登録済みのアドオンについて、GitHub Releasesを再取得して最新状態に同期する
// @Tags         admin
// @Produce      json
// @Param        id path string true "addon id"
// @Success      200 {object} addon.SyncResult
// @Failure      404 {object} errorResponse
// @Failure      502 {object} errorResponse
// @Router       /admin/addons/{id}/refresh [post]
func RefreshAddonHandler(store *addon.Store, gh *github.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		a, err := store.GetAddonByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, addon.ErrNotFound) {
				writeError(w, http.StatusNotFound, "addon not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		result, err := addon.SyncAddon(r.Context(), gh, store, a.GithubOwner, a.GithubRepo)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

type adminAddonsResponse struct {
	Addons []addon.AddonWithVersions `json:"addons"`
}

// ListAdminAddonsHandler godoc
// @Summary      登録済みアドオン一覧(管理用)
// @Description  登録済みの全アドオンとそのバージョンを返す
// @Tags         admin
// @Produce      json
// @Success      200 {object} adminAddonsResponse
// @Router       /admin/addons [get]
func ListAdminAddonsHandler(store *addon.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addons, err := store.ListAddons(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, adminAddonsResponse{Addons: addons})
	}
}
