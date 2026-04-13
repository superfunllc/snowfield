package snowfield

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportManifestOmitsDatasetVersionWhenNoOverride(t *testing.T) {
	loaded := loadTestDataset(t)
	outputDir := t.TempDir()

	if _, err := Export(loaded, outputDir, "2026-04-13T07:56:35Z", ""); err != nil {
		t.Fatalf("Export: %v", err)
	}

	manifest := readManifest(t, outputDir, "full")
	if _, present := manifest["dataset_version"]; present {
		t.Fatalf("expected dataset_version to be absent, got %v", manifest["dataset_version"])
	}
}

func TestExportManifestUsesDatasetVersionOverride(t *testing.T) {
	loaded := loadTestDataset(t)
	outputDir := t.TempDir()
	releaseTag := "snow-fields-20260413T075635Z-13-3326293b766e"

	if _, err := Export(loaded, outputDir, "2026-04-13T07:56:35Z", releaseTag); err != nil {
		t.Fatalf("Export: %v", err)
	}

	for _, variant := range []string{"full", "local"} {
		manifest := readManifest(t, outputDir, variant)
		if got := manifest["dataset_version"]; got != releaseTag {
			t.Fatalf("%s manifest dataset_version: got %v want %q", variant, got, releaseTag)
		}
	}
}

func TestExportRejectsInvalidDatasetVersionOverride(t *testing.T) {
	loaded := loadTestDataset(t)

	_, err := Export(loaded, t.TempDir(), "2026-04-13T07:56:35Z", "Snow fields snow-fields-20260413T075635Z-13-3326293b766e")
	if err == nil {
		t.Fatal("expected invalid dataset version override error")
	}
	if !strings.Contains(err.Error(), "dataset version override") {
		t.Fatalf("unexpected error: %v", err)
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
