package addon

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Addon is a registered GitHub repository this server tracks releases for.
type Addon struct {
	ID          string    `json:"id"`
	GithubOwner string    `json:"github_owner"`
	GithubRepo  string    `json:"github_repo"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Version is a single GitHub release of an Addon, along with whatever we
// managed to extract from its BP pack's scripts/properties.js.
type Version struct {
	ID               string          `json:"id"`
	AddonID          string          `json:"addon_id"`
	GithubReleaseID  int64           `json:"github_release_id"`
	TagName          string          `json:"tag_name"`
	ZipAssetName     *string         `json:"zip_asset_name"`
	ZipAssetURL      *string         `json:"zip_asset_url"`
	PublishedAt      time.Time       `json:"published_at"`
	Properties       json.RawMessage `json:"properties" swaggertype:"object"`
	PropertiesError  *string         `json:"properties_error"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// AddonWithVersions bundles an Addon with all of its known versions, newest
// release first.
type AddonWithVersions struct {
	Addon
	Versions []Version `json:"versions"`
}

// Store is the Postgres-backed persistence layer for addons and their
// versions. It takes a *pgxpool.Pool directly, matching the rest of this
// codebase's style (see internal/api/db_health.go) rather than introducing
// a repository interface nothing else needs yet.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore builds a Store backed by pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// UpsertAddon creates the addon if it doesn't exist yet, or touches its
// updated_at if it does, keyed on (github_owner, github_repo).
func (s *Store) UpsertAddon(ctx context.Context, owner, repo string) (Addon, error) {
	const q = `
		INSERT INTO addons (github_owner, github_repo)
		VALUES ($1, $2)
		ON CONFLICT (github_owner, github_repo) DO UPDATE SET updated_at = now()
		RETURNING id, github_owner, github_repo, created_at, updated_at`

	var a Addon
	err := s.pool.QueryRow(ctx, q, owner, repo).Scan(&a.ID, &a.GithubOwner, &a.GithubRepo, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return Addon{}, fmt.Errorf("upsert addon %s/%s: %w", owner, repo, err)
	}
	return a, nil
}

// ErrNotFound is returned by lookups that find no matching row.
var ErrNotFound = fmt.Errorf("not found")

// GetAddonByID looks up a single addon by its UUID.
func (s *Store) GetAddonByID(ctx context.Context, id string) (Addon, error) {
	const q = `
		SELECT id, github_owner, github_repo, created_at, updated_at
		FROM addons WHERE id = $1`

	var a Addon
	err := s.pool.QueryRow(ctx, q, id).Scan(&a.ID, &a.GithubOwner, &a.GithubRepo, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Addon{}, ErrNotFound
		}
		return Addon{}, fmt.Errorf("get addon %s: %w", id, err)
	}
	return a, nil
}

// GetAddonByOwnerRepo looks up a single addon by its GitHub owner/repo.
func (s *Store) GetAddonByOwnerRepo(ctx context.Context, owner, repo string) (Addon, error) {
	const q = `
		SELECT id, github_owner, github_repo, created_at, updated_at
		FROM addons WHERE github_owner = $1 AND github_repo = $2`

	var a Addon
	err := s.pool.QueryRow(ctx, q, owner, repo).Scan(&a.ID, &a.GithubOwner, &a.GithubRepo, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Addon{}, ErrNotFound
		}
		return Addon{}, fmt.Errorf("get addon %s/%s: %w", owner, repo, err)
	}
	return a, nil
}

// UpsertVersion creates or updates a single release row, keyed on
// (addon_id, github_release_id).
func (s *Store) UpsertVersion(ctx context.Context, v Version) error {
	const q = `
		INSERT INTO addon_versions
			(addon_id, github_release_id, tag_name, zip_asset_name, zip_asset_url, published_at, properties, properties_error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (addon_id, github_release_id) DO UPDATE SET
			tag_name = EXCLUDED.tag_name,
			zip_asset_name = EXCLUDED.zip_asset_name,
			zip_asset_url = EXCLUDED.zip_asset_url,
			published_at = EXCLUDED.published_at,
			properties = EXCLUDED.properties,
			properties_error = EXCLUDED.properties_error,
			updated_at = now()`

	var props interface{}
	if v.Properties != nil {
		props = v.Properties
	}

	_, err := s.pool.Exec(ctx, q,
		v.AddonID, v.GithubReleaseID, v.TagName, v.ZipAssetName, v.ZipAssetURL, v.PublishedAt, props, v.PropertiesError)
	if err != nil {
		return fmt.Errorf("upsert version %s (release %d): %w", v.TagName, v.GithubReleaseID, err)
	}
	return nil
}

// ListAddons returns every registered addon along with its versions
// (newest release first).
func (s *Store) ListAddons(ctx context.Context) ([]AddonWithVersions, error) {
	const addonsQ = `SELECT id, github_owner, github_repo, created_at, updated_at FROM addons ORDER BY github_owner, github_repo`

	rows, err := s.pool.Query(ctx, addonsQ)
	if err != nil {
		return nil, fmt.Errorf("list addons: %w", err)
	}
	defer rows.Close()

	var result []AddonWithVersions
	for rows.Next() {
		var a Addon
		if err := rows.Scan(&a.ID, &a.GithubOwner, &a.GithubRepo, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan addon: %w", err)
		}
		result = append(result, AddonWithVersions{Addon: a})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list addons: %w", err)
	}

	for i := range result {
		versions, err := s.ListVersions(ctx, result[i].ID)
		if err != nil {
			return nil, err
		}
		result[i].Versions = versions
	}

	return result, nil
}

// ListVersions returns every known version of a single addon, newest
// release first.
func (s *Store) ListVersions(ctx context.Context, addonID string) ([]Version, error) {
	const q = `
		SELECT id, addon_id, github_release_id, tag_name, zip_asset_name, zip_asset_url,
			published_at, properties, properties_error, created_at, updated_at
		FROM addon_versions
		WHERE addon_id = $1
		ORDER BY published_at DESC`

	rows, err := s.pool.Query(ctx, q, addonID)
	if err != nil {
		return nil, fmt.Errorf("list versions for addon %s: %w", addonID, err)
	}
	defer rows.Close()

	var versions []Version
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list versions for addon %s: %w", addonID, err)
	}

	return versions, nil
}

// GetVersionByTag looks up a single version by its owner/repo/tag, used by
// the public download redirect endpoint.
func (s *Store) GetVersionByTag(ctx context.Context, owner, repo, tag string) (Version, error) {
	const q = `
		SELECT v.id, v.addon_id, v.github_release_id, v.tag_name, v.zip_asset_name, v.zip_asset_url,
			v.published_at, v.properties, v.properties_error, v.created_at, v.updated_at
		FROM addon_versions v
		JOIN addons a ON a.id = v.addon_id
		WHERE a.github_owner = $1 AND a.github_repo = $2 AND v.tag_name = $3`

	row := s.pool.QueryRow(ctx, q, owner, repo, tag)
	v, err := scanVersion(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Version{}, ErrNotFound
		}
		return Version{}, fmt.Errorf("get version %s/%s@%s: %w", owner, repo, tag, err)
	}
	return v, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanVersion(row rowScanner) (Version, error) {
	var v Version
	err := row.Scan(&v.ID, &v.AddonID, &v.GithubReleaseID, &v.TagName, &v.ZipAssetName, &v.ZipAssetURL,
		&v.PublishedAt, &v.Properties, &v.PropertiesError, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return Version{}, err
	}
	return v, nil
}
