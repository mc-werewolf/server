package api

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	worlddata "github.com/mc-werewolf/server/backend/internal/world"
)

const maxWorldUploadBytes = 2 << 30 // 2 GiB

var worldVersionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func worldReleaseResponse(release worlddata.Release) worlddata.Release {
	release.DownloadURL = "/api/world/latest/download"
	return release
}

func CurrentWorldHandler(store *worlddata.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		release, err := store.Current(r.Context())
		if errors.Is(err, worlddata.ErrNotFound) {
			writeError(w, http.StatusNotFound, "world is not configured")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, worldReleaseResponse(release))
	}
}

func UploadWorldHandler(store *worlddata.Store, storageDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxWorldUploadBytes)
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeError(w, http.StatusBadRequest, "invalid upload or file is too large")
			return
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}

		version := strings.TrimSpace(r.FormValue("version"))
		if !worldVersionPattern.MatchString(version) {
			writeError(w, http.StatusBadRequest, "version must be 1-64 letters, numbers, dots, underscores, or hyphens")
			return
		}

		source, header, err := r.FormFile("world")
		if err != nil {
			writeError(w, http.StatusBadRequest, "world file is required")
			return
		}
		defer source.Close()
		originalName := filepath.Base(header.Filename)
		if ext := strings.ToLower(filepath.Ext(originalName)); ext != ".mcworld" && ext != ".zip" {
			writeError(w, http.StatusBadRequest, "world must be a .mcworld or .zip file")
			return
		}

		if err := os.MkdirAll(storageDir, 0o750); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to prepare world storage")
			return
		}
		temp, err := os.CreateTemp(storageDir, ".upload-*.mcworld")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create upload file")
			return
		}
		tempPath := temp.Name()
		defer os.Remove(tempPath)

		hash := sha256.New()
		size, copyErr := io.Copy(io.MultiWriter(temp, hash), source)
		closeErr := temp.Close()
		if copyErr != nil || closeErr != nil || size == 0 {
			writeError(w, http.StatusBadRequest, "failed to read world file")
			return
		}
		if err := worlddata.ValidateArchive(tempPath); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		digest := fmt.Sprintf("%x", hash.Sum(nil))
		storedName := digest + ".mcworld"
		storedPath := filepath.Join(storageDir, storedName)
		if err := os.Rename(tempPath, storedPath); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to store world file")
			return
		}

		release, err := store.SetCurrent(r.Context(), worlddata.Release{
			Version: version, OriginalName: originalName, StoredName: storedName,
			FileSize: size, SHA256: digest,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, worldReleaseResponse(release))
	}
}

func DownloadWorldHandler(store *worlddata.Store, storageDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		release, err := store.Current(r.Context())
		if errors.Is(err, worlddata.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, release.OriginalName))
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("ETag", `"`+release.SHA256+`"`)
		http.ServeFile(w, r, filepath.Join(storageDir, release.StoredName))
	}
}
