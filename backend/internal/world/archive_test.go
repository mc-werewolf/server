package world

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeArchive(t *testing.T, names ...string) string {
	t.Helper()
	filePath := filepath.Join(t.TempDir(), "world.mcworld")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, name := range names {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte("data")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return filePath
}

func TestValidateArchive(t *testing.T) {
	if err := ValidateArchive(writeArchive(t, "level.dat", "db/000001.log")); err != nil {
		t.Fatalf("valid archive rejected: %v", err)
	}
}

func TestValidateArchiveRequiresLevelDat(t *testing.T) {
	if err := ValidateArchive(writeArchive(t, "db/000001.log")); err == nil {
		t.Fatal("archive without level.dat was accepted")
	}
}

func TestValidateArchiveRejectsTraversal(t *testing.T) {
	if err := ValidateArchive(writeArchive(t, "level.dat", "../secret")); err == nil {
		t.Fatal("archive with traversal was accepted")
	}
}
