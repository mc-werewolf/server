package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	gameNetwork "github.com/mc-werewolf/server/backend/internal/network"
)

type networkServerStore interface {
	Register(context.Context, gameNetwork.RegisterInput) (gameNetwork.Registration, error)
	Heartbeat(context.Context, string, string, gameNetwork.HeartbeatInput) (gameNetwork.Server, error)
	Stop(context.Context, string, string) error
	ListActive(context.Context) ([]gameNetwork.Server, error)
}

type registerServerRequest struct {
	DisplayName string `json:"displayName"`
	WorldName   string `json:"worldName"`
	MaxPlayers  int    `json:"maxPlayers"`
}

type heartbeatServerRequest struct {
	PlayerCount    int     `json:"playerCount"`
	MaxPlayers     int     `json:"maxPlayers"`
	Status         string  `json:"status"`
	HostName       *string `json:"hostName"`
	HostPort       *int    `json:"hostPort"`
	ConnectionMode string  `json:"connectionMode"`
}

func RegisterNetworkServerHandler(store networkServerStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request registerServerRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		request.DisplayName = strings.TrimSpace(request.DisplayName)
		request.WorldName = strings.TrimSpace(request.WorldName)
		if !validLabel(request.DisplayName) || !validLabel(request.WorldName) || request.MaxPlayers < 1 || request.MaxPlayers > 100 {
			writeError(w, http.StatusBadRequest, "displayName/worldName must be 1-80 characters and maxPlayers must be 1-100")
			return
		}
		registration, err := store.Register(r.Context(), gameNetwork.RegisterInput{
			DisplayName: request.DisplayName,
			WorldName:   request.WorldName,
			MaxPlayers:  request.MaxPlayers,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to register game server")
			return
		}
		writeJSON(w, http.StatusCreated, registration)
	}
}

func HeartbeatNetworkServerHandler(store networkServerStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "bearer token is required")
			return
		}
		var request heartbeatServerRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := validateHeartbeat(request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if request.ConnectionMode == "pending" {
			request.HostName = nil
			request.HostPort = nil
		}
		server, err := store.Heartbeat(r.Context(), r.PathValue("id"), token, gameNetwork.HeartbeatInput{
			PlayerCount: request.PlayerCount, MaxPlayers: request.MaxPlayers,
			Status: request.Status, HostName: request.HostName, HostPort: request.HostPort,
			ConnectionMode: request.ConnectionMode,
		})
		if errors.Is(err, gameNetwork.ErrNotFound) {
			writeError(w, http.StatusNotFound, "game server not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update game server")
			return
		}
		writeJSON(w, http.StatusOK, server)
	}
}

func StopNetworkServerHandler(store networkServerStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "bearer token is required")
			return
		}
		err := store.Stop(r.Context(), r.PathValue("id"), token)
		if errors.Is(err, gameNetwork.ErrNotFound) {
			writeError(w, http.StatusNotFound, "game server not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to stop game server")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func ListNetworkServersHandler(store networkServerStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servers, err := store.ListActive(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list game servers")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"servers": servers})
	}
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 16*1024))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func bearerToken(r *http.Request) (string, bool) {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	token, found := strings.CutPrefix(value, "Bearer ")
	return token, found && len(token) >= 32
}

func validLabel(value string) bool { return len(value) >= 1 && len(value) <= 80 }

func validateHeartbeat(request heartbeatServerRequest) error {
	if request.PlayerCount < 0 || request.MaxPlayers < 1 || request.MaxPlayers > 100 || request.PlayerCount > request.MaxPlayers {
		return errors.New("invalid player counts")
	}
	if request.Status != "starting" && request.Status != "online" {
		return errors.New("status must be starting or online")
	}
	if request.ConnectionMode != "pending" && request.ConnectionMode != "direct" && request.ConnectionMode != "relay" {
		return errors.New("connectionMode must be pending, direct, or relay")
	}
	if request.ConnectionMode == "pending" {
		return nil
	}
	if request.HostName == nil || !validHostName(*request.HostName) || request.HostPort == nil || *request.HostPort < 1 || *request.HostPort > 65535 {
		return errors.New("direct/relay servers require a valid hostName IP and hostPort")
	}
	return nil
}

func validHostName(value string) bool {
	if net.ParseIP(value) != nil {
		return true
	}
	if len(value) < 1 || len(value) > 253 {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) < 1 || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, character := range label {
			if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '-') {
				return false
			}
		}
	}
	return true
}
