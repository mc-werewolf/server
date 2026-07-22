package world

import (
	"archive/zip"
	"fmt"
	"path"
	"strings"
)

// ValidateArchive rejects path traversal and requires a Bedrock level.dat.
func ValidateArchive(filePath string) error {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return fmt.Errorf("invalid .mcworld archive: %w", err)
	}
	defer reader.Close()

	hasLevelDat := false
	for _, entry := range reader.File {
		name := strings.ReplaceAll(entry.Name, "\\", "/")
		clean := path.Clean(name)
		if name == "" || strings.HasPrefix(name, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("archive contains an unsafe path: %q", entry.Name)
		}
		if path.Base(clean) == "level.dat" {
			hasLevelDat = true
		}
	}
	if !hasLevelDat {
		return fmt.Errorf("archive does not contain level.dat")
	}
	return nil
}
