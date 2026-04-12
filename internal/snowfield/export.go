package snowfield

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func Export(loaded *Loaded, outputDir string, generatedAt string) ([]string, error) {
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}

	var written []string
	for _, variant := range []string{"full", "local"} {
		rows, err := rowsForVariant(loaded, variant)
		if err != nil {
			return nil, err
		}

		csvPath := filepath.Join(outputDir, fmt.Sprintf("snow_fields.%s.csv", variant))
		geoJSONPath := filepath.Join(outputDir, fmt.Sprintf("snow_fields.%s.geojson", variant))
		minJSONPath := filepath.Join(outputDir, fmt.Sprintf("snow_fields.%s.min.json", variant))
		manifestPath := filepath.Join(outputDir, fmt.Sprintf("snow_fields.%s.manifest.json", variant))

		if err := writeCSV(csvPath, rows, loaded.Catalog.FieldsWithFlag("csv")); err != nil {
			return nil, err
		}
		if err := writeMinJSON(minJSONPath, rows, loaded.Catalog.FieldsWithFlag("min_json")); err != nil {
			return nil, err
		}
		geocodedRowCount, err := writeGeoJSON(geoJSONPath, rows, loaded.Catalog.FieldsWithFlag("csv"))
		if err != nil {
			return nil, err
		}
		if err := writeManifest(manifestPath, loaded.Dataset, variant, generatedAt, len(rows), geocodedRowCount, csvPath, geoJSONPath, minJSONPath); err != nil {
			return nil, err
		}

		written = append(written, csvPath, geoJSONPath, minJSONPath, manifestPath)
	}

	return written, nil
}

func rowsForVariant(loaded *Loaded, variant string) ([]Record, error) {
	switch variant {
	case "full":
		return loaded.Dataset.Records, nil
	case "local":
		localRegions := loaded.Catalog.LocalRegions()
		rows := make([]Record, 0, len(loaded.Dataset.Records))
		for _, record := range loaded.Dataset.Records {
			if regions, ok := localRegions[record.CountryCode]; ok {
				if _, ok := regions[record.RegionCode]; ok {
					rows = append(rows, record)
				}
			}
		}
		return rows, nil
	default:
		return nil, fmt.Errorf("unknown variant %q", variant)
	}
}

func writeCSV(path string, rows []Record, fields []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write(fields); err != nil {
		return err
	}
	for _, row := range rows {
		values, err := recordMap(row)
		if err != nil {
			return err
		}
		csvRow := make([]string, 0, len(fields))
		for _, field := range fields {
			csvRow = append(csvRow, csvValue(values[field]))
		}
		if err := writer.Write(csvRow); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeMinJSON(path string, rows []Record, fields []string) error {
	payload := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		values, err := recordMap(row)
		if err != nil {
			return err
		}
		rowPayload := make(map[string]any, len(fields))
		for _, field := range fields {
			rowPayload[field] = values[field]
		}
		payload = append(payload, rowPayload)
	}
	return writeJSONFile(path, payload)
}

func writeGeoJSON(path string, rows []Record, propertyFields []string) (int, error) {
	features := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if row.Lat == nil || row.Lng == nil {
			continue
		}
		values, err := recordMap(row)
		if err != nil {
			return 0, err
		}
		properties := make(map[string]any, len(propertyFields))
		for _, field := range propertyFields {
			if field == "lat" || field == "lng" {
				continue
			}
			properties[field] = values[field]
		}
		features = append(features, map[string]any{
			"type": "Feature",
			"id":   row.CatalogID,
			"geometry": map[string]any{
				"type":        "Point",
				"coordinates": []float64{*row.Lng, *row.Lat},
			},
			"properties": properties,
		})
	}

	payload := map[string]any{
		"type":     "FeatureCollection",
		"features": features,
	}
	if err := writeJSONFile(path, payload); err != nil {
		return 0, err
	}
	return len(features), nil
}

func writeManifest(path string, dataset Dataset, variant string, generatedAt string, rowCount int, geocodedRowCount int, csvPath string, geoJSONPath string, minJSONPath string) error {
	csvSHA, err := fileSHA256(csvPath)
	if err != nil {
		return err
	}
	geoJSONSHA, err := fileSHA256(geoJSONPath)
	if err != nil {
		return err
	}
	minJSONSHA, err := fileSHA256(minJSONPath)
	if err != nil {
		return err
	}

	manifest := map[string]any{
		"dataset_name":       dataset.DatasetName,
		"dataset_version":    dataset.DatasetVersion,
		"schema_version":     dataset.SchemaVersion,
		"variant":            variant,
		"generated_at":       generatedAt,
		"row_count":          rowCount,
		"geocoded_row_count": geocodedRowCount,
		"sha256":             csvSHA,
		"assets": map[string]any{
			"csv": map[string]any{
				"path":      filepath.Base(csvPath),
				"sha256":    csvSHA,
				"row_count": rowCount,
			},
			"geojson": map[string]any{
				"path":          filepath.Base(geoJSONPath),
				"sha256":        geoJSONSHA,
				"feature_count": geocodedRowCount,
			},
			"min_json": map[string]any{
				"path":      filepath.Base(minJSONPath),
				"sha256":    minJSONSHA,
				"row_count": rowCount,
			},
		},
	}
	return writeJSONFile(path, manifest)
}

func writeJSONFile(path string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func csvValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return fmt.Sprint(typed)
	}
}
