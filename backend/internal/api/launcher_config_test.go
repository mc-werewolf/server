package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewLauncherConfigBuildsKairoLatestURLs(t *testing.T) {
	config := NewLauncherConfig("https://kairojs.com/", []string{"Kairo", " game-manager ", "kairo", "../invalid"})
	if config.SchemaVersion != 1 || config.RegistryURL != "https://kairojs.com" || len(config.Addons) != 2 {
		t.Fatalf("config = %+v", config)
	}
	if config.Addons[0].LatestVersionURL != "https://kairojs.com/api/v1/addons/kairo/versions/latest" {
		t.Fatalf("addon = %+v", config.Addons[0])
	}
}

func TestLauncherConfigHandler(t *testing.T) {
	handler := LauncherConfigHandler(NewLauncherConfig("https://kairojs.com", []string{"game-manager"}))
	response := httptest.NewRecorder()
	handler(response, httptest.NewRequest(http.MethodGet, "/api/launcher/v1/config", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":"game-manager"`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "public, max-age=300" {
		t.Fatalf("cache-control = %q", response.Header().Get("Cache-Control"))
	}
}

func TestAdditionalRolesIsOptional(t *testing.T) {
	config := NewLauncherConfig("https://kairojs.com", []string{"additional-roles-1"})
	if len(config.Addons) != 1 || config.Addons[0].Required {
		t.Fatalf("addons = %+v", config.Addons)
	}
}
