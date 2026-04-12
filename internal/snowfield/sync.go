package snowfield

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type SyncOptions struct {
	Variant        string
	SchemaMode     string
	Table          string
	DryRun         bool
	SupabaseURL    string
	ServiceRoleKey string
	Stdout         io.Writer
}

func Sync(loaded *Loaded, options SyncOptions) (int, error) {
	rows, err := rowsForVariant(loaded, options.Variant)
	if err != nil {
		return 0, err
	}

	columns, conflictColumn, err := loaded.Catalog.SyncColumns(options.SchemaMode)
	if err != nil {
		return 0, err
	}

	payload := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		values, err := recordMap(row)
		if err != nil {
			return 0, err
		}
		payloadRow := make(map[string]any, len(columns))
		for _, column := range columns {
			payloadRow[column] = values[column]
		}
		payload = append(payload, payloadRow)
	}

	if options.DryRun {
		out := options.Stdout
		if out == nil {
			out = io.Discard
		}
		dryRunPayload := map[string]any{
			"table":       options.Table,
			"on_conflict": conflictColumn,
			"rows":        payload,
		}
		data, err := json.MarshalIndent(dryRunPayload, "", "  ")
		if err != nil {
			return 0, err
		}
		fmt.Fprintln(out, string(data))
		return len(payload), nil
	}

	if options.SupabaseURL == "" || options.ServiceRoleKey == "" {
		return 0, fmt.Errorf("SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY are required unless --dry-run is set")
	}

	endpoint := strings.TrimRight(options.SupabaseURL, "/") + "/rest/v1/" + url.PathEscape(options.Table)
	endpoint += "?on_conflict=" + url.QueryEscape(conflictColumn)
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	request.Header.Set("apikey", options.ServiceRoleKey)
	request.Header.Set("Authorization", "Bearer "+options.ServiceRoleKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Prefer", "resolution=merge-duplicates,return=minimal")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		return 0, fmt.Errorf("supabase status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return len(payload), nil
}
