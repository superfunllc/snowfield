package snowfield

import (
	"encoding/json"
	"fmt"
	"os"
)

type Loaded struct {
	Dataset    Dataset
	Catalog    FieldCatalog
	RawDataset map[string]any
	RawRecords []map[string]any
}

type Dataset struct {
	Schema         string   `json:"$schema,omitempty"`
	DatasetName   string   `json:"dataset_name"`
	SchemaVersion int      `json:"schema_version"`
	Description    string   `json:"description,omitempty"`
	Records        []Record `json:"records"`
}

type Record struct {
	CatalogID         string   `json:"catalog_id"`
	Slug              string   `json:"slug"`
	Source            string   `json:"source"`
	SourceID          string   `json:"source_id"`
	Name              string   `json:"name"`
	CountryCode       string   `json:"country_code"`
	RegionCode        string   `json:"region_code"`
	RegionName        string   `json:"region_name"`
	Locality          string   `json:"locality"`
	Timezone          string   `json:"timezone"`
	Lat               *float64 `json:"lat"`
	Lng               *float64 `json:"lng"`
	ElevationFT       *int     `json:"elevation_ft"`
	BaseElevationFT   *int     `json:"base_elevation_ft"`
	SummitElevationFT *int     `json:"summit_elevation_ft"`
	VerticalDropFT    *int     `json:"vertical_drop_ft"`
	Status            string   `json:"status"`
	IsActive          bool     `json:"is_active"`
	IsVerified        bool     `json:"is_verified"`
	Tags              []string `json:"tags"`
	UpdatedAt         string   `json:"updated_at"`
	Sources           []Source `json:"sources"`
}

type Source struct {
	Type        string  `json:"type"`
	Name        string  `json:"name"`
	URL         *string `json:"url"`
	RetrievedAt string  `json:"retrieved_at"`
	Note        string  `json:"note,omitempty"`
}

func Load(datasetPath string, schemaPath string) (*Loaded, error) {
	catalog, err := LoadFieldCatalog(schemaPath)
	if err != nil {
		return nil, err
	}

	datasetBytes, err := os.ReadFile(datasetPath)
	if err != nil {
		return nil, fmt.Errorf("read dataset: %w", err)
	}

	var dataset Dataset
	if err := json.Unmarshal(datasetBytes, &dataset); err != nil {
		return nil, fmt.Errorf("decode dataset: %w", err)
	}

	var rawDataset map[string]any
	if err := json.Unmarshal(datasetBytes, &rawDataset); err != nil {
		return nil, fmt.Errorf("decode raw dataset: %w", err)
	}

	rawRecordValues, ok := rawDataset["records"].([]any)
	if !ok {
		rawRecordValues = nil
	}
	rawRecords := make([]map[string]any, 0, len(rawRecordValues))
	for index, value := range rawRecordValues {
		record, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("records[%d]: expected object", index)
		}
		rawRecords = append(rawRecords, record)
	}

	return &Loaded{
		Dataset:    dataset,
		Catalog:    catalog,
		RawDataset: rawDataset,
		RawRecords: rawRecords,
	}, nil
}

func recordMap(record Record) (map[string]any, error) {
	recordBytes, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	var values map[string]any
	if err := json.Unmarshal(recordBytes, &values); err != nil {
		return nil, err
	}
	return values, nil
}
