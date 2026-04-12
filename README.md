# snowfield

Canonical Snowpool snow field dataset.

This repo follows the Git-first dataset strategy:

- `data/snow_fields.json` is the editable source of truth.
- Supabase/Postgres is a deployed runtime copy, not the master record.
- CSV, GeoJSON, minified JSON, and manifests are generated release artifacts.
- CI validates the dataset and uploads generated artifacts.
- Supabase sync is guarded behind manual workflow input until Snowpool's catalog schema is ready.

## Quick Start

```sh
make validate
make export
```

Generated files are written to `dist/`:

- `snow_fields.full.csv`
- `snow_fields.full.geojson`
- `snow_fields.full.min.json`
- `snow_fields.full.manifest.json`
- `snow_fields.local.csv`
- `snow_fields.local.geojson`
- `snow_fields.local.min.json`
- `snow_fields.local.manifest.json`

## Bootstrap Data

The initial records come from Snowpool's current canonical migration-backed list:

- Alta
- Brighton
- Heavenly
- Kirkwood
- Northstar
- Palisades Tahoe
- Snowbird

Only fields already present in Snowpool are treated as verified. Unknown coordinates and detailed base/summit/vertical fields are left `null` until sourced.

## Supabase Sync

For the current Snowpool table shape:

```sh
SUPABASE_URL=... SUPABASE_SERVICE_ROLE_KEY=... make sync-legacy
```

For the future expanded catalog schema:

```sh
SUPABASE_URL=... SUPABASE_SERVICE_ROLE_KEY=... make sync-catalog
```

## When the Snowpool Schema Changes

Treat Snowpool's database schema and this dataset repo as separate contracts:

- The app repo owns the Supabase/Postgres table shape through migrations.
- Snowfield owns the canonical catalog content, validation rules, generated artifacts, and import payload.
- Migrations should not become the long-term owner of canonical snow field rows.

When `snow_fields` changes in Snowpool:

1. Add the Supabase migration in the app repo.
2. Decide whether the new column is canonical catalog data or app/runtime metadata.
3. If it is canonical catalog data, add the field once in `schema/snow_fields.schema.json`.
   The schema's `required` list controls required fields; its `x-snowfield` metadata controls CSV export fields, minified JSON fields, Supabase sync columns, conflict keys, and local variant region rules.
4. Add values for the new field in `data/snow_fields.json`.
5. Update `docs/DATASET_CONTRACT.md` when the public data contract changes.
6. Update scripts only when the field needs custom validation, transform, or sync behavior that is not covered by the schema metadata.
7. Bump `schema_version` when the dataset release contract changes in a breaking way.
8. Regenerate and verify artifacts:

```sh
make validate
make export
python3 scripts/sync_supabase.py --dry-run --schema-mode catalog --variant full
```

If a new DB column is required, make sure the migration provides a default/backfill or the dataset has complete values before running a real sync.

See `docs/DATASET_CONTRACT.md` for the field contract and release model.
