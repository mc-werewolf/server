package github

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseRepoURL extracts the {owner}/{repo} pair from any URL under a GitHub
// repository, discarding anything beyond the repo itself (releases, tree,
// blob, issues, etc.). The scheme may be omitted (e.g. "github.com/o/r").
func ParseRepoURL(raw string) (owner, repo string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("url is empty")
	}

	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("parse url: %w", err)
	}

	host := strings.ToLower(u.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return "", "", fmt.Errorf("not a github.com url: %q", raw)
	}

	var segments []string
	for _, s := range strings.Split(u.Path, "/") {
		if s != "" {
			segments = append(segments, s)
		}
	}
	if len(segments) < 2 {
		return "", "", fmt.Errorf("url does not contain an owner/repo path: %q", raw)
	}

	owner = segments[0]
	repo = strings.TrimSuffix(segments[1], ".git")
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("url does not contain an owner/repo path: %q", raw)
	}

	return owner, repo, nil
}
