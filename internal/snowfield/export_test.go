package snowfield

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExportManifestDatasetVersionIsContentHash(t *testing.T) {
	loaded := loadTestDataset(t)
	outputDir := t.TempDir()

	if _, err := Export(loaded, outputDir, "2026-04-13T07:56:35Z"); err != nil {
		t.Fatalf("Export: %v", err)
	}

	for _, variant := range []string{"full", "local"} {
		manifest := readManifest(t, outputDir, variant)
		got, present := manifest["dataset_version"]
		if !present {
			t.Fatalf("%s manifest: dataset_version missing", variant)
		}
		if got != loaded.DatasetHash {
			t.Fatalf("%s manifest dataset_version: got %v want %q", variant, got, loaded.DatasetHash)
		}
	}
}

func loadTestDataset(t *testing.T) *Loaded {
	t.Helper()

	loaded, err := Load("../../data/snow_fields.json", "../../schema/snow_fields.schema.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return loaded
}

func readManifest(t *testing.T, outputDir string, variant string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(outputDir, "snow_fields."+variant+".manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	return manifest
}
