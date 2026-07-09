package addon

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

// buildZip builds an in-memory zip from a map of entry name -> content.
func buildZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("create entry %q: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("write entry %q: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

const samplePropertiesJS = `var properties = { id: "kairo" };
export {
  properties
};
`

func TestExtractBPPropertiesJS_TopLevel(t *testing.T) {
	bpZip := buildZip(t, map[string]string{
		"scripts/properties.js": samplePropertiesJS,
	})
	outerZip := buildZip(t, map[string]string{
		"Kairo_BP.zip": string(bpZip),
		"Kairo_RP.zip": "not a real RP zip, irrelevant",
	})

	got, err := ExtractBPPropertiesJS(outerZip)
	if err != nil {
		t.Fatalf("ExtractBPPropertiesJS() unexpected error: %v", err)
	}
	if got != samplePropertiesJS {
		t.Fatalf("ExtractBPPropertiesJS() = %q, want %q", got, samplePropertiesJS)
	}
}

func TestExtractBPPropertiesJS_McpackAndNestedFolder(t *testing.T) {
	bpZip := buildZip(t, map[string]string{
		"kairo_root/scripts/properties.js": samplePropertiesJS,
	})
	outerZip := buildZip(t, map[string]string{
		"Kairo_BP.mcpack": string(bpZip),
		"Kairo_RP.mcpack": "not a real RP pack, irrelevant",
	})

	got, err := ExtractBPPropertiesJS(outerZip)
	if err != nil {
		t.Fatalf("ExtractBPPropertiesJS() unexpected error: %v", err)
	}
	if got != samplePropertiesJS {
		t.Fatalf("ExtractBPPropertiesJS() = %q, want %q", got, samplePropertiesJS)
	}
}

func TestExtractBPPropertiesJS_NoBPEntry(t *testing.T) {
	outerZip := buildZip(t, map[string]string{
		"Kairo_RP.zip": "irrelevant",
	})

	_, err := ExtractBPPropertiesJS(outerZip)
	if err == nil {
		t.Fatal("ExtractBPPropertiesJS() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "find BP pack") {
		t.Fatalf("ExtractBPPropertiesJS() error = %v, want mention of BP pack", err)
	}
}

func TestExtractBPPropertiesJS_NoPropertiesJS(t *testing.T) {
	bpZip := buildZip(t, map[string]string{
		"manifest.json": "{}",
	})
	outerZip := buildZip(t, map[string]string{
		"Kairo_BP.zip": string(bpZip),
	})

	_, err := ExtractBPPropertiesJS(outerZip)
	if err == nil {
		t.Fatal("ExtractBPPropertiesJS() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "properties.js") {
		t.Fatalf("ExtractBPPropertiesJS() error = %v, want mention of properties.js", err)
	}
}

func TestExtractBPPropertiesJS_InvalidOuterZip(t *testing.T) {
	_, err := ExtractBPPropertiesJS([]byte("not a zip"))
	if err == nil {
		t.Fatal("ExtractBPPropertiesJS() expected error, got nil")
	}
}
