package world

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("world release not found")

type Release struct {
	Version      string    `json:"version"`
	OriginalName string    `json:"fileName"`
	FileSize     int64     `json:"fileSize"`
	SHA256       string    `json:"sha256"`
	UpdatedAt    time.Time `json:"updatedAt"`
	DownloadURL  string    `json:"downloadUrl"`
	StoredName   string    `json:"-"`
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Current(ctx context.Context) (Release, error) {
	const query = `SELECT version, file_name, original_name, file_size, sha256, updated_at
		FROM world_release WHERE singleton_id = 1`
	var release Release
	err := s.pool.QueryRow(ctx, query).Scan(&release.Version, &release.StoredName, &release.OriginalName, &release.FileSize, &release.SHA256, &release.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Release{}, ErrNotFound
	}
	if err != nil {
		return Release{}, fmt.Errorf("get current world: %w", err)
	}
	return release, nil
}

func (s *Store) SetCurrent(ctx context.Context, release Release) (Release, error) {
	const query = `INSERT INTO world_release
		(singleton_id, version, file_name, original_name, file_size, sha256)
		VALUES (1, $1, $2, $3, $4, $5)
		ON CONFLICT (singleton_id) DO UPDATE SET
			version = EXCLUDED.version,
			file_name = EXCLUDED.file_name,
			original_name = EXCLUDED.original_name,
			file_size = EXCLUDED.file_size,
			sha256 = EXCLUDED.sha256,
			updated_at = now()
		RETURNING updated_at`
	if err := s.pool.QueryRow(ctx, query, release.Version, release.StoredName, release.OriginalName, release.FileSize, release.SHA256).Scan(&release.UpdatedAt); err != nil {
		return Release{}, fmt.Errorf("set current world: %w", err)
	}
	return release, nil
}
