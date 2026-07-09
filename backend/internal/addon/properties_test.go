package addon

import (
	"encoding/json"
	"testing"
)

// kairoPropertiesJS is the confirmed real-world sample this project targets:
// esbuild-bundled output of src/properties.ts.
const kairoPropertiesJS = `// src/properties.ts
var properties = {
  id: "kairo",
  //# // a-z & 0-9 - _
  metadata: {
    authors: ["shizuku86"]
  },
  header: {
    name: "Kairo",
    description: "Enables communication between multiple behavior packs by leveraging the ScriptAPI as a communication layer.",
    version: {
      major: 1,
      minor: 0,
      patch: 0,
      prerelease: "beta.3"
    },
    min_engine_version: { major: 1, minor: 21, patch: 132 }
  },
  minecraftDependencies: [
    {
      module_name: "@minecraft/server",
      version: "2.8.0"
    },
    {
      module_name: "@minecraft/server-ui",
      version: "2.0.0"
    }
  ],
  optionalDependencies: {
    "kairo-database": "^1.0.0"
  },
  dependencies: {
    /**
     * id: version (string) // "kairo": "1.0.0"
     */
  },
  tags: ["official", "stable"]
};
export {
  properties
};
`

func TestParsePropertiesJS_KairoSample(t *testing.T) {
	raw, err := ParsePropertiesJS(kairoPropertiesJS)
	if err != nil {
		t.Fatalf("ParsePropertiesJS() unexpected error: %v", err)
	}

	var got struct {
		ID     string `json:"id"`
		Header struct {
			Name    string `json:"name"`
			Version struct {
				Major      int    `json:"major"`
				Minor      int    `json:"minor"`
				Patch      int    `json:"patch"`
				Prerelease string `json:"prerelease"`
			} `json:"version"`
		} `json:"header"`
		MinecraftDependencies []struct {
			ModuleName string `json:"module_name"`
			Version    string `json:"version"`
		} `json:"minecraftDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
		Tags                 []string          `json:"tags"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal extracted properties: %v", err)
	}

	if got.ID != "kairo" {
		t.Errorf("ID = %q, want %q", got.ID, "kairo")
	}
	if got.Header.Name != "Kairo" {
		t.Errorf("Header.Name = %q, want %q", got.Header.Name, "Kairo")
	}
	if got.Header.Version.Major != 1 || got.Header.Version.Minor != 0 || got.Header.Version.Patch != 0 {
		t.Errorf("Header.Version = %+v, want 1.0.0", got.Header.Version)
	}
	if got.Header.Version.Prerelease != "beta.3" {
		t.Errorf("Header.Version.Prerelease = %q, want %q", got.Header.Version.Prerelease, "beta.3")
	}
	if len(got.MinecraftDependencies) != 2 || got.MinecraftDependencies[0].ModuleName != "@minecraft/server" {
		t.Errorf("MinecraftDependencies = %+v, want 2 entries starting with @minecraft/server", got.MinecraftDependencies)
	}
	if got.OptionalDependencies["kairo-database"] != "^1.0.0" {
		t.Errorf("OptionalDependencies[kairo-database] = %q, want %q", got.OptionalDependencies["kairo-database"], "^1.0.0")
	}
	if len(got.Tags) != 2 || got.Tags[0] != "official" || got.Tags[1] != "stable" {
		t.Errorf("Tags = %v, want [official stable]", got.Tags)
	}
}

func TestParsePropertiesJS_MissingPropertiesVar(t *testing.T) {
	_, err := ParsePropertiesJS(`var somethingElse = { id: "x" };`)
	if err == nil {
		t.Fatal("ParsePropertiesJS() expected error, got nil")
	}
}

func TestParsePropertiesJS_SyntaxError(t *testing.T) {
	_, err := ParsePropertiesJS(`var properties = {`)
	if err == nil {
		t.Fatal("ParsePropertiesJS() expected error, got nil")
	}
}
