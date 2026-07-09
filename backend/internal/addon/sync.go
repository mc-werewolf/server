package addon

import (
	"context"
	"fmt"

	"github.com/mc-werewolf/server/backend/internal/github"
)

// SyncResult summarizes the outcome of a SyncAddon call.
type SyncResult struct {
	Addon          AddonWithVersions `json:"addon"`
	VersionsSynced int               `json:"versions_synced"`
	VersionErrors  int               `json:"version_errors"`
}

// SyncAddon registers (or re-syncs) owner/repo: it upserts the addon row,
// fetches every non-draft release from GitHub, and for each one attempts to
// extract the BP pack's properties.js. A release that fails at any step
// (no .zip asset, download failure, malformed BP pack, missing/broken
// properties.js) still gets a row written, with the failure reason captured
// in PropertiesError -- one broken release must never block the rest of the
// sync.
func SyncAddon(ctx context.Context, gh *github.Client, store *Store, owner, repo string) (SyncResult, error) {
	a, err := store.UpsertAddon(ctx, owner, repo)
	if err != nil {
		return SyncResult{}, err
	}

	releases, err := gh.ListReleases(ctx, owner, repo)
	if err != nil {
		return SyncResult{}, fmt.Errorf("list releases for %s/%s: %w", owner, repo, err)
	}

	result := SyncResult{}
	for _, release := range releases {
		if release.Draft {
			continue
		}

		v := Version{
			AddonID:         a.ID,
			GithubReleaseID: release.ID,
			TagName:         release.TagName,
			PublishedAt:     release.PublishedAt,
		}

		if err := populateVersion(ctx, gh, &v, release); err != nil {
			errMsg := err.Error()
			v.PropertiesError = &errMsg
			result.VersionErrors++
		} else {
			result.VersionsSynced++
		}

		if err := store.UpsertVersion(ctx, v); err != nil {
			return SyncResult{}, err
		}
	}

	versions, err := store.ListVersions(ctx, a.ID)
	if err != nil {
		return SyncResult{}, err
	}
	result.Addon = AddonWithVersions{Addon: a, Versions: versions}

	return result, nil
}

// populateVersion fills in v's asset/properties fields from the release's
// .zip asset. Any returned error means v should still be persisted, just
// with PropertiesError set instead of Properties.
func populateVersion(ctx context.Context, gh *github.Client, v *Version, release github.Release) error {
	asset, err := SelectZipAsset(release.Assets)
	if err != nil {
		return err
	}
	v.ZipAssetName = &asset.Name
	v.ZipAssetURL = &asset.BrowserDownloadURL

	zipBytes, err := gh.DownloadAsset(ctx, asset.BrowserDownloadURL)
	if err != nil {
		return err
	}

	propsJS, err := ExtractBPPropertiesJS(zipBytes)
	if err != nil {
		return err
	}

	props, err := ParsePropertiesJS(propsJS)
	if err != nil {
		return err
	}

	v.Properties = props
	return nil
}
