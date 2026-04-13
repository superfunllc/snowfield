package snowfield

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

var datasetVersionPattern = regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2}([-.+][A-Za-z0-9_.-]+)?$`)
var catalogIDPattern = regexp.MustCompile(`^[a-z0-9_.:-]+$`)
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var sourcePattern = regexp.MustCompile(`^[a-z0-9_]+$`)
var countryCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)
var tagPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

const verticalDropToleranceFT = 50

var statusActiveValue = map[string]bool{
	"active":   true,
	"proposed": false,
	"inactive": false,
	"retired":  false,
}

func Validate(loaded *Loaded) []string {
	var errors []string
	dataset := loaded.Dataset

	if dataset.DatasetName != "snow_fields" {
		errors = append(errors, "dataset_name: expected 'snow_fields'")
	}
	if !datasetVersionPattern.MatchString(dataset.DatasetVersion) {
		errors = append(errors, "dataset_version: expected YYYY.MM.DD or YYYY.MM.DD-suffix")
	}
	if dataset.SchemaVersion != 2 {
		errors = append(errors, "schema_version: expected 2")
	}
	if len(dataset.Records) == 0 {
		errors = append(errors, "records: expected at least one record")
		return errors
	}

	if rawRecordsValue, ok := loaded.RawDataset["records"]; !ok || rawRecordsValue == nil {
		errors = append(errors, "records: expected list")
	}

	catalogIDs := make(map[string]struct{}, len(dataset.Records))
	slugs := make(map[string]struct{}, len(dataset.Records))
	sourceKeys := make(map[string]struct{}, len(dataset.Records))
	regionalNames := make(map[string]struct{}, len(dataset.Records))
	sortedCatalogIDs := make([]string, 0, len(dataset.Records))
	knownFields := loaded.Catalog.Defs.SnowField.Properties

	for index, record := range dataset.Records {
		path := fmt.Sprintf("records[%d]", index)
		if index >= len(loaded.RawRecords) {
			errors = append(errors, fmt.Sprintf("%s: missing raw record", path))
			continue
		}
		rawRecord := loaded.RawRecords[index]

		for _, field := range loaded.Catalog.RecordFields() {
			if _, ok := rawRecord[field]; !ok {
				errors = append(errors, fmt.Sprintf("%s.%s: missing required field", path, field))
			}
		}
		for field := range rawRecord {
			if _, ok := knownFields[field]; !ok {
				errors = append(errors, fmt.Sprintf("%s.%s: unknown field", path, field))
			}
		}

		if !catalogIDPattern.MatchString(record.CatalogID) {
			errors = append(errors, fmt.Sprintf("%s.catalog_id: expected lowercase stable id", path))
		} else {
			if _, exists := catalogIDs[record.CatalogID]; exists {
				errors = append(errors, fmt.Sprintf("%s.catalog_id: duplicate %q", path, record.CatalogID))
			}
			catalogIDs[record.CatalogID] = struct{}{}
			sortedCatalogIDs = append(sortedCatalogIDs, record.CatalogID)
		}

		if !slugPattern.MatchString(record.Slug) {
			errors = append(errors, fmt.Sprintf("%s.slug: expected lowercase hyphenated slug", path))
		} else {
			if _, exists := slugs[record.Slug]; exists {
				errors = append(errors, fmt.Sprintf("%s.slug: duplicate %q", path, record.Slug))
			}
			slugs[record.Slug] = struct{}{}
		}

		if !sourcePattern.MatchString(record.Source) {
			errors = append(errors, fmt.Sprintf("%s.source: expected lowercase source id", path))
		}
		if strings.TrimSpace(record.SourceID) == "" {
			errors = append(errors, fmt.Sprintf("%s.source_id: expected non-empty string", path))
		}
		sourceKey := record.Source + "\x00" + record.SourceID
		if _, exists := sourceKeys[sourceKey]; exists {
			errors = append(errors, fmt.Sprintf("%s.source_id: duplicate source key (%q, %q)", path, record.Source, record.SourceID))
		}
		sourceKeys[sourceKey] = struct{}{}

		if strings.TrimSpace(record.Name) == "" {
			errors = append(errors, fmt.Sprintf("%s.name: expected non-empty string", path))
		}
		if !countryCodePattern.MatchString(record.CountryCode) {
			errors = append(errors, fmt.Sprintf("%s.country_code: expected ISO 3166-1 alpha-2 code", path))
		}
		if strings.TrimSpace(record.RegionCode) == "" {
			errors = append(errors, fmt.Sprintf("%s.region_code: expected non-empty string", path))
		}
		regionalName := record.CountryCode + "\x00" + record.RegionCode + "\x00" + strings.ToLower(record.Name)
		if _, exists := regionalNames[regionalName]; exists {
			errors = append(errors, fmt.Sprintf("%s.name: duplicate name within country/region %q", path, record.Name))
		}
		regionalNames[regionalName] = struct{}{}

		for _, field := range []struct {
			name  string
			value string
		}{
			{"region_name", record.RegionName},
			{"locality", record.Locality},
			{"timezone", record.Timezone},
		} {
			if strings.TrimSpace(field.value) == "" {
				errors = append(errors, fmt.Sprintf("%s.%s: expected non-empty string", path, field.name))
			}
		}

		validateNullableFloat(record.Lat, path+".lat", -90, 90, &errors)
		validateNullableFloat(record.Lng, path+".lng", -180, 180, &errors)
		if (record.Lat == nil) != (record.Lng == nil) {
			errors = append(errors, fmt.Sprintf("%s: lat and lng must either both be set or both be null", path))
		}

		validateNullableInt(record.ElevationFT, path+".elevation_ft", -2000, 30000, &errors)
		validateNullableInt(record.BaseElevationFT, path+".base_elevation_ft", -2000, 30000, &errors)
		validateNullableInt(record.SummitElevationFT, path+".summit_elevation_ft", -2000, 30000, &errors)
		validateNullableInt(record.VerticalDropFT, path+".vertical_drop_ft", 0, 30000, &errors)
		if record.BaseElevationFT != nil && record.SummitElevationFT != nil && *record.BaseElevationFT > *record.SummitElevationFT {
			errors = append(errors, fmt.Sprintf("%s: base_elevation_ft must be <= summit_elevation_ft", path))
		}
		if record.BaseElevationFT != nil && record.SummitElevationFT != nil && record.VerticalDropFT != nil {
			delta := *record.SummitElevationFT - *record.BaseElevationFT - *record.VerticalDropFT
			if absInt(delta) > verticalDropToleranceFT {
				errors = append(errors, fmt.Sprintf("%s: vertical_drop_ft must be within %d ft of summit_elevation_ft - base_elevation_ft", path, verticalDropToleranceFT))
			}
		}

		expectedIsActive, ok := statusActiveValue[record.Status]
		if !ok {
			statuses := make([]string, 0, len(statusActiveValue))
			for status := range statusActiveValue {
				statuses = append(statuses, status)
			}
			sort.Strings(statuses)
			errors = append(errors, fmt.Sprintf("%s.status: expected one of %v", path, statuses))
		} else if record.IsActive != expectedIsActive {
			errors = append(errors, fmt.Sprintf("%s: is_active must be %t when status is %q", path, expectedIsActive, record.Status))
		}

		seenTags := make(map[string]struct{}, len(record.Tags))
		for tagIndex, tag := range record.Tags {
			tagPath := fmt.Sprintf("%s.tags[%d]", path, tagIndex)
			if !tagPattern.MatchString(tag) {
				errors = append(errors, fmt.Sprintf("%s: expected lowercase tag", tagPath))
				continue
			}
			if _, exists := seenTags[tag]; exists {
				errors = append(errors, fmt.Sprintf("%s: duplicate tag %q", tagPath, tag))
			}
			seenTags[tag] = struct{}{}
		}
		if !sort.StringsAreSorted(record.Tags) {
			errors = append(errors, fmt.Sprintf("%s.tags: tags must be sorted", path))
		}

		validateDate(record.UpdatedAt, path+".updated_at", &errors)
		validateSources(record.Sources, rawRecord["sources"], index, loaded.Catalog.SourceFields(), &errors)
	}

	if !sort.StringsAreSorted(sortedCatalogIDs) {
		errors = append(errors, "records: must be sorted by catalog_id")
	}

	return errors
}

func validateNullableFloat(value *float64, path string, minValue float64, maxValue float64, errors *[]string) {
	if value == nil {
		return
	}
	if *value < minValue || *value > maxValue {
		*errors = append(*errors, fmt.Sprintf("%s: %v outside expected range %v..%v", path, *value, minValue, maxValue))
	}
}

func validateNullableInt(value *int, path string, minValue int, maxValue int, errors *[]string) {
	if value == nil {
		return
	}
	if *value < minValue || *value > maxValue {
		*errors = append(*errors, fmt.Sprintf("%s: %v outside expected range %v..%v", path, *value, minValue, maxValue))
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func validateDate(value string, path string, errors *[]string) {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		*errors = append(*errors, fmt.Sprintf("%s: invalid date %q; expected YYYY-MM-DD", path, value))
	}
}

func validateSources(sources []Source, rawSourcesValue any, recordIndex int, requiredFields []string, errors *[]string) {
	path := fmt.Sprintf("records[%d].sources", recordIndex)
	rawSources, ok := rawSourcesValue.([]any)
	if !ok {
		*errors = append(*errors, fmt.Sprintf("%s: expected list", path))
		return
	}
	if len(sources) == 0 || len(rawSources) == 0 {
		*errors = append(*errors, fmt.Sprintf("%s: expected non-empty list", path))
		return
	}

	for sourceIndex, source := range sources {
		sourcePath := fmt.Sprintf("%s[%d]", path, sourceIndex)
		if sourceIndex >= len(rawSources) {
			*errors = append(*errors, fmt.Sprintf("%s: missing raw source", sourcePath))
			continue
		}
		rawSource, ok := rawSources[sourceIndex].(map[string]any)
		if !ok {
			*errors = append(*errors, fmt.Sprintf("%s: expected object", sourcePath))
			continue
		}
		for _, field := range requiredFields {
			if _, ok := rawSource[field]; !ok {
				*errors = append(*errors, fmt.Sprintf("%s.%s: missing required field", sourcePath, field))
			}
		}
		if strings.TrimSpace(source.Type) == "" {
			*errors = append(*errors, fmt.Sprintf("%s.type: expected non-empty string", sourcePath))
		}
		if strings.TrimSpace(source.Name) == "" {
			*errors = append(*errors, fmt.Sprintf("%s.name: expected non-empty string", sourcePath))
		}
		validateDate(source.RetrievedAt, sourcePath+".retrieved_at", errors)
	}
}
