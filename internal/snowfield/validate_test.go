package snowfield

import (
	"slices"
	"testing"
)

func TestValidateRejectsTopLevelDatasetVersion(t *testing.T) {
	loaded := loadTestDataset(t)
	loaded.RawDataset["dataset_version"] = "stale"

	errors := Validate(loaded)
	if !slices.Contains(errors, "dataset_version: unknown field") {
		t.Fatalf("Validate errors: got %v want dataset_version unknown field error", errors)
	}
}
