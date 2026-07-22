package api

import (
	"net/http"
	"regexp"
	"strings"
)

var launcherAddonIDPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

type LauncherAddon struct {
	ID               string `json:"id"`
	Required         bool   `json:"required"`
	LatestVersionURL string `json:"latestVersionUrl"`
}

type LauncherConfig struct {
	SchemaVersion int             `json:"schemaVersion"`
	RegistryURL   string          `json:"registryUrl"`
	Addons        []LauncherAddon `json:"addons"`
}

func NewLauncherConfig(registryURL string, addonIDs []string) LauncherConfig {
	registryURL = strings.TrimRight(registryURL, "/")
	addons := make([]LauncherAddon, 0, len(addonIDs))
	seen := make(map[string]struct{}, len(addonIDs))
	for _, rawID := range addonIDs {
		id := strings.ToLower(strings.TrimSpace(rawID))
		if !launcherAddonIDPattern.MatchString(id) {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		addons = append(addons, LauncherAddon{
			ID:               id,
			Required:         id != "additional-roles-1",
			LatestVersionURL: registryURL + "/api/v1/addons/" + id + "/versions/latest",
		})
	}
	return LauncherConfig{SchemaVersion: 1, RegistryURL: registryURL, Addons: addons}
}

func LauncherConfigHandler(config LauncherConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=300")
		writeJSON(w, http.StatusOK, config)
	}
}
