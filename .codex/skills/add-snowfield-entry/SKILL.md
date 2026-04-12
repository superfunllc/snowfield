---
name: add-snowfield-entry
description: Add or update a snowfield record in this snowfield dataset repo. Use when the user asks to add a new ski resort/snowfield entry, update canonical snowfield data, add sourced coordinates/elevation metadata, or validate/export dataset changes in data/snow_fields.json.
---

# Add Snowfield Entry

## Workflow

1. Read `README.md`, `schema/snow_fields.schema.json`, and the nearby records in `data/snow_fields.json` before editing.
2. Add or update the record in `data/snow_fields.json` only. Do not edit generated files under `dist/`.
3. Choose stable identity fields:
   - `catalog_id`: permanent internal dataset ID, for example `snowpool:us-ca-new-resort`
   - `slug`: app/API-facing stable slug, for example `us-ca-new-resort`
   - `source` plus `source_id`: import dedupe key
4. Fill only sourced facts. Leave unchecked coordinates and detailed elevation values as `null`; do not guess.
5. Include at least one `sources` entry with `type`, `name`, `url`, and `retrieved_at`.
6. Keep `records` sorted by `catalog_id`.
7. Run:

```sh
make validate
make export
make sync-catalog-dry-run
```

8. Review the diff. Normally commit `data/snow_fields.json` and any docs/schema/tooling changes that were intentionally needed; do not commit ignored `dist/` artifacts.

## Schema Changes

If the requested entry needs a new canonical field, add the field once in `schema/snow_fields.schema.json` and use its `x-snowfield` metadata for CSV, minified JSON, and Supabase sync behavior. Update the Go CLI only when the field needs custom validation, transformation, or sync behavior not covered by schema metadata.
