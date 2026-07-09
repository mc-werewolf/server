package addon

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// maxEntrySize caps how much of a single zip entry we will decompress into
// memory, as a safety net against maliciously/accidentally huge entries.
const maxEntrySize = 200 * 1024 * 1024 // 200MB

// ExtractBPPropertiesJS takes the bytes of a release's outer .zip asset,
// finds the nested BP pack (a "*BP.zip" or "*BP.mcpack" entry -- .mcpack is
// just a renamed .zip, so it's opened the same way regardless of
// extension), and returns the raw text of its scripts/properties.js file.
//
// It is a pure function: no network or DB access, which makes it cheap to
// test with in-memory zip fixtures.
func ExtractBPPropertiesJS(outerZip []byte) (string, error) {
	outer, err := zip.NewReader(bytes.NewReader(outerZip), int64(len(outerZip)))
	if err != nil {
		return "", fmt.Errorf("open outer zip: %w", err)
	}

	bpEntry, err := findBySuffix(outer.File, "bp.zip", "bp.mcpack")
	if err != nil {
		return "", fmt.Errorf("find BP pack in outer zip: %w", err)
	}

	bpBytes, err := readZipEntry(bpEntry)
	if err != nil {
		return "", fmt.Errorf("read BP pack: %w", err)
	}

	bp, err := zip.NewReader(bytes.NewReader(bpBytes), int64(len(bpBytes)))
	if err != nil {
		return "", fmt.Errorf("open BP pack as zip: %w", err)
	}

	propsEntry, err := findBySuffix(bp.File, "scripts/properties.js")
	if err != nil {
		return "", fmt.Errorf("find scripts/properties.js in BP pack: %w", err)
	}

	propsBytes, err := readZipEntry(propsEntry)
	if err != nil {
		return "", fmt.Errorf("read scripts/properties.js: %w", err)
	}

	return string(propsBytes), nil
}

// findBySuffix returns the first file whose (lower-cased) name ends with
// any of the given suffixes.
func findBySuffix(files []*zip.File, suffixes ...string) (*zip.File, error) {
	for _, f := range files {
		name := strings.ToLower(f.Name)
		for _, suffix := range suffixes {
			if strings.HasSuffix(name, suffix) {
				return f, nil
			}
		}
	}
	return nil, fmt.Errorf("no entry found matching suffix(es) %v", suffixes)
}

func readZipEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open entry %q: %w", f.Name, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(io.LimitReader(rc, maxEntrySize+1))
	if err != nil {
		return nil, fmt.Errorf("read entry %q: %w", f.Name, err)
	}
	if len(data) > maxEntrySize {
		return nil, fmt.Errorf("entry %q exceeds max size of %d bytes", f.Name, maxEntrySize)
	}

	return data, nil
}
