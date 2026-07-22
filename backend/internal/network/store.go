package network

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const LeaseDuration = 90 * time.Second

var ErrNotFound = errors.New("game server not found")

type Server struct {
	ID             string    `json:"id"`
	DisplayName    string    `json:"displayName"`
	WorldName      string    `json:"worldName"`
	HostName       *string   `json:"hostName,omitempty"`
	HostPort       *int      `json:"hostPort,omitempty"`
	ConnectionMode string    `json:"connectionMode"`
	PlayerCount    int       `json:"playerCount"`
	MaxPlayers     int       `json:"maxPlayers"`
	Status         string    `json:"status"`
	LeaseExpiresAt time.Time `json:"leaseExpiresAt"`
}

type Registration struct {
	Server
	Token string `json:"token"`
}

type RegisterInput struct {
	DisplayName string
	WorldName   string
	MaxPlayers  int
}

type HeartbeatInput struct {
	PlayerCount    int
	MaxPlayers     int
	Status         string
	HostName       *string
	HostPort       *int
	ConnectionMode string
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Register(ctx context.Context, input RegisterInput) (Registration, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return Registration{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	hash := sha256.Sum256([]byte(token))
	lease := time.Now().UTC().Add(LeaseDuration)
	var server Server
	err := s.pool.QueryRow(ctx, `
		INSERT INTO game_servers
			(token_hash, display_name, world_name, max_players, lease_expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, display_name, world_name, host_name, host_port,
			connection_mode, player_count, max_players, status, lease_expires_at
	`, hash[:], input.DisplayName, input.WorldName, input.MaxPlayers, lease).Scan(
		&server.ID, &server.DisplayName, &server.WorldName, &server.HostName,
		&server.HostPort, &server.ConnectionMode, &server.PlayerCount,
		&server.MaxPlayers, &server.Status, &server.LeaseExpiresAt,
	)
	return Registration{Server: server, Token: token}, err
}

func (s *Store) Heartbeat(ctx context.Context, id, token string, input HeartbeatInput) (Server, error) {
	hash := sha256.Sum256([]byte(token))
	lease := time.Now().UTC().Add(LeaseDuration)
	var server Server
	err := s.pool.QueryRow(ctx, `
		UPDATE game_servers SET
			player_count = $3, max_players = $4, status = $5,
			host_name = $6, host_port = $7, connection_mode = $8,
			lease_expires_at = $9, updated_at = now()
		WHERE id = $1 AND token_hash = $2 AND status <> 'stopping'
		RETURNING id, display_name, world_name, host_name, host_port,
			connection_mode, player_count, max_players, status, lease_expires_at
	`, id, hash[:], input.PlayerCount, input.MaxPlayers, input.Status,
		input.HostName, input.HostPort, input.ConnectionMode, lease).Scan(
		&server.ID, &server.DisplayName, &server.WorldName, &server.HostName,
		&server.HostPort, &server.ConnectionMode, &server.PlayerCount,
		&server.MaxPlayers, &server.Status, &server.LeaseExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Server{}, ErrNotFound
	}
	return server, err
}

func (s *Store) Stop(ctx context.Context, id, token string) error {
	hash := sha256.Sum256([]byte(token))
	result, err := s.pool.Exec(ctx, `
		UPDATE game_servers SET status = 'stopping', lease_expires_at = now(), updated_at = now()
		WHERE id = $1 AND token_hash = $2
	`, id, hash[:])
	if err == nil && result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) ListActive(ctx context.Context) ([]Server, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, display_name, world_name, host_name, host_port,
			connection_mode, player_count, max_players, status, lease_expires_at
		FROM game_servers
		WHERE lease_expires_at > now() AND status = 'online'
		ORDER BY player_count DESC, updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	servers := make([]Server, 0)
	for rows.Next() {
		var server Server
		if err := rows.Scan(&server.ID, &server.DisplayName, &server.WorldName,
			&server.HostName, &server.HostPort, &server.ConnectionMode,
			&server.PlayerCount, &server.MaxPlayers, &server.Status,
			&server.LeaseExpiresAt); err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, rows.Err()
}
