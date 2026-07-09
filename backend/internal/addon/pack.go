// Package addon contains the domain logic for syncing Minecraft addon
// releases: picking the right release asset, digging into the nested
// zip/BP/RP structure, evaluating scripts/properties.js, and persisting the
// results.
package addon

import (
	"fmt"
	"strings"

	"github.com/mc-werewolf/server/backend/internal/github"
)

// SelectZipAsset picks the single ".zip" asset out of a release's assets,
// ignoring any ".mcaddon" (or other) assets.
func SelectZipAsset(assets []github.Asset) (github.Asset, error) {
	var matches []github.Asset
	for _, a := range assets {
		if strings.HasSuffix(strings.ToLower(a.Name), ".zip") {
			matches = append(matches, a)
		}
	}

	switch len(matches) {
	case 0:
		return github.Asset{}, fmt.Errorf("no .zip asset found")
	case 1:
		return matches[0], nil
	default:
		return github.Asset{}, fmt.Errorf("expected exactly one .zip asset, found %d", len(matches))
	}
}
