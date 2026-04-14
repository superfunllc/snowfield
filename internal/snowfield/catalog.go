package snowfield

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type FieldCatalog struct {
	XSnowfield CatalogMetadata            `json:"x-snowfield"`
	Properties map[string]json.RawMessage `json:"properties"`
	Defs       CatalogDefs                `json:"$defs"`
}

type CatalogMetadata struct {
	LocalRegions map[string][]string `json:"local_regions"`
}

type CatalogDefs struct {
	Source    SourceDef    `json:"source"`
	SnowField SnowFieldDef `json:"snow_field"`
}

type SourceDef struct {
	Required []string `json:"required"`
}

type SnowFieldDef struct {
	Required   []string               `json:"required"`
	Properties map[string]FieldSchema `json:"properties"`
}

type FieldSchema struct {
	XSnowfield FieldMetadata `json:"x-snowfield"`
}

type FieldMetadata struct {
	CSV        bool `json:"csv"`
	ClientJSON bool `json:"client_json"`
}

func LoadFieldCatalog(path string) (FieldCatalog, error) {
	schemaBytes, err := os.ReadFile(path)
	if err != nil {
		return FieldCatalog{}, fmt.Errorf("read field catalog: %w", err)
	}

	var catalog FieldCatalog
	if err := json.Unmarshal(schemaBytes, &catalog); err != nil {
		return FieldCatalog{}, fmt.Errorf("decode field catalog: %w", err)
	}
	if len(catalog.Defs.SnowField.Required) == 0 {
		return FieldCatalog{}, fmt.Errorf("field catalog missing $defs.snow_field.required")
	}
	if len(catalog.Defs.SnowField.Properties) == 0 {
		return FieldCatalog{}, fmt.Errorf("field catalog missing $defs.snow_field.properties")
	}
	if len(catalog.Properties) == 0 {
		return FieldCatalog{}, fmt.Errorf("field catalog missing properties")
	}
	return catalog, nil
}

func (c FieldCatalog) RecordFields() []string {
	return append([]string(nil), c.Defs.SnowField.Required...)
}

func (c FieldCatalog) SourceFields() []string {
	return append([]string(nil), c.Defs.Source.Required...)
}

func (c FieldCatalog) FieldsWithFlag(flag string) []string {
	fields := make([]string, 0)
	for _, field := range c.orderedFields() {
		metadata := c.Defs.SnowField.Properties[field].XSnowfield
		if flag == "csv" && metadata.CSV {
			fields = append(fields, field)
		}
		if flag == "client_json" && metadata.ClientJSON {
			fields = append(fields, field)
		}
	}
	return fields
}

func (c FieldCatalog) LocalRegions() map[string]map[string]struct{} {
	regions := make(map[string]map[string]struct{}, len(c.XSnowfield.LocalRegions))
	for countryCode, regionCodes := range c.XSnowfield.LocalRegions {
		regions[countryCode] = make(map[string]struct{}, len(regionCodes))
		for _, regionCode := range regionCodes {
			regions[countryCode][regionCode] = struct{}{}
		}
	}
	return regions
}

func (c FieldCatalog) orderedFields() []string {
	fields := append([]string(nil), c.Defs.SnowField.Required...)
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		seen[field] = struct{}{}
	}

	optionalFields := make([]string, 0)
	for field := range c.Defs.SnowField.Properties {
		if _, ok := seen[field]; !ok {
			optionalFields = append(optionalFields, field)
		}
	}
	sort.Strings(optionalFields)
	fields = append(fields, optionalFields...)
	return fields
}
