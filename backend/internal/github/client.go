// Package github provides a minimal client for the parts of the GitHub REST
// API this server needs: listing a repository's releases and downloading a
// release asset. It has no knowledge of addons/BP/RP packs — that logic
// lives in internal/addon.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

// maxAssetSize caps how much of a release asset we will read into memory.
// Minecraft addon packs are small (single-digit MB); this is a generous
// safety net against a misconfigured/huge asset.
const maxAssetSize = 200 * 1024 * 1024 // 200MB

var nextLinkRe = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

// Asset is a single file attached to a GitHub release.
type Asset struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Release is a single GitHub release.
type Release struct {
	ID          int64     `json:"id"`
	TagName     string    `json:"tag_name"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Client talks to the public GitHub REST API without authentication.
type Client struct {
	httpClient *http.Client
}

// NewClient builds a Client with a sane request timeout.
func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

// ListReleases returns every release (including prereleases, excluding
// nothing) for owner/repo, paginating through GitHub's Link header.
func (c *Client) ListReleases(ctx context.Context, owner, repo string) ([]Release, error) {
	var all []Release
	next := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=100", owner, repo)

	for next != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, next, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		req.Header.Set("User-Agent", "mc-werewolf-server")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list releases for %s/%s: %w", owner, repo, err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			return nil, fmt.Errorf("list releases for %s/%s: unexpected status %d: %s", owner, repo, resp.StatusCode, body)
		}

		var page []Release
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode releases for %s/%s: %w", owner, repo, err)
		}
		resp.Body.Close()

		all = append(all, page...)
		next = nextPageURL(resp.Header.Get("Link"))
	}

	return all, nil
}

func nextPageURL(linkHeader string) string {
	m := nextLinkRe.FindStringSubmatch(linkHeader)
	if m == nil {
		return ""
	}
	return m[1]
}

// DownloadAsset fetches the given (already-resolved, e.g. browser_download_url)
// asset URL entirely into memory. It never touches disk.
func (c *Client) DownloadAsset(ctx context.Context, assetURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "mc-werewolf-server")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download asset %s: %w", assetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download asset %s: unexpected status %d", assetURL, resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAssetSize+1))
	if err != nil {
		return nil, fmt.Errorf("read asset %s: %w", assetURL, err)
	}
	if len(data) > maxAssetSize {
		return nil, fmt.Errorf("asset %s exceeds max size of %d bytes", assetURL, maxAssetSize)
	}

	return data, nil
}
