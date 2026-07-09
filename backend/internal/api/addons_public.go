package api

import (
	"errors"
	"net/http"

	"github.com/mc-werewolf/server/backend/internal/addon"
)

type publicAddonsResponse struct {
	Addons []addon.AddonWithVersions `json:"addons"`
}

// ListAddonsHandler godoc
// @Summary      アドオン一覧
// @Description  登録済みの全アドオンとそのバージョンを返す(bds-launcher等の公開API)
// @Tags         addons
// @Produce      json
// @Success      200 {object} publicAddonsResponse
// @Router       /addons [get]
func ListAddonsHandler(store *addon.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addons, err := store.ListAddons(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, publicAddonsResponse{Addons: addons})
	}
}

type addonVersionsResponse struct {
	Versions []addon.Version `json:"versions"`
}

// ListAddonVersionsHandler godoc
// @Summary      アドオンのバージョン一覧
// @Description  指定したowner/repoのアドオンについて、既知の全バージョンとproperties.jsから抽出したメタ情報を返す
// @Tags         addons
// @Produce      json
// @Param        owner path string true "GitHub owner"
// @Param        repo path string true "GitHub repo"
// @Success      200 {object} addonVersionsResponse
// @Failure      404 {object} errorResponse
// @Router       /addons/{owner}/{repo}/versions [get]
func ListAddonVersionsHandler(store *addon.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner := r.PathValue("owner")
		repo := r.PathValue("repo")

		a, err := store.GetAddonByOwnerRepo(r.Context(), owner, repo)
		if err != nil {
			if errors.Is(err, addon.ErrNotFound) {
				writeError(w, http.StatusNotFound, "addon not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		versions, err := store.ListVersions(r.Context(), a.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, addonVersionsResponse{Versions: versions})
	}
}

// DownloadAddonVersionHandler godoc
// @Summary      アドオンバージョンのダウンロード
// @Description  指定バージョンのGitHubリリースアセット(.zip)へリダイレクトする。実ファイルはGitHub上のものをそのまま使う
// @Tags         addons
// @Param        owner path string true "GitHub owner"
// @Param        repo path string true "GitHub repo"
// @Param        tag path string true "release tag"
// @Success      302
// @Failure      404 {object} errorResponse
// @Router       /addons/{owner}/{repo}/versions/{tag}/download [get]
func DownloadAddonVersionHandler(store *addon.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner := r.PathValue("owner")
		repo := r.PathValue("repo")
		tag := r.PathValue("tag")

		v, err := store.GetVersionByTag(r.Context(), owner, repo, tag)
		if err != nil {
			if errors.Is(err, addon.ErrNotFound) {
				writeError(w, http.StatusNotFound, "version not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if v.ZipAssetURL == nil {
			writeError(w, http.StatusNotFound, "no downloadable asset for this version")
			return
		}

		http.Redirect(w, r, *v.ZipAssetURL, http.StatusFound)
	}
}
