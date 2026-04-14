# snowfield

Canonical Snowpool snow field dataset.

This repo follows the Git-first dataset strategy:

- `data/snow_fields.json` is the editable source of truth.
- Supabase/Postgres is a deployed runtime copy, not the master record.
- CSV, GeoJSON, client JSON, and manifests are generated release artifacts.
- CI validates PRs and publishes generated artifacts as GitHub releases after `main` updates.

## Quick Start

```sh
make validate
make export
```

Generated files are written to `dist/`:

- `snow_fields.full.csv`
- `snow_fields.full.geojson`
- `snow_fields.full.client.json`
- `snow_fields.full.manifest.json`
- `snow_fields.local.csv`
- `snow_fields.local.geojson`
- `snow_fields.local.client.json`
- `snow_fields.local.manifest.json`

To add a new snowfield entry:

1. Edit `data/snow_fields.json` and add a new object under `records`.
2. Choose stable identifiers:
   - `catalog_id`: permanent internal dataset ID, for example `snowpool:us-ca-new-resort`
   - `slug`: app/API-facing stable slug, for example `us-ca-new-resort`
   - `source` and `source_id`: import dedupe key
3. Fill only sourced facts. Leave unchecked coordinates and detailed elevation values as `null`; do not guess.
4. Keep records sorted by `catalog_id`.
5. Add at least one source provenance entry with `type`, `name`, `url`, and `retrieved_at`.
6. Run:

```sh
make validate
make export
```

Review the diff before committing. `dist/` is generated and ignored, so new snowfield entries normally only change `data/snow_fields.json`.

## Dataset Contract

`data/snow_fields.json` is the editable source of truth. Generated files in `dist/` are release artifacts; generate them with `make export` instead of editing them by hand.

Identity rules:

- `catalog_id` is the permanent repo-owned identity for a snow field.
- `slug` is the stable app-facing identifier for APIs and clients.
- `source` plus `source_id` is the import key for source-specific deduplication.
- Display `name` is not an identity. Names can change and can collide across regions.

Versioning rules:

- Release manifests always include `dataset_version` as the SHA-256 content hash of `data/snow_fields.json`. Do not edit `dataset_version` into the canonical JSON.
- `schema_version` tracks the dataset contract independently. Bump it only when the contract changes in a breaking way.

Required fields are defined by `schema/snow_fields.schema.json` under `$defs.snow_field.required`. The same schema is also the shared field catalog for the Go CLI. Its `x-snowfield` metadata controls:

- CSV export fields
- client JSON fields
- local variant region rules

Coordinates and detailed elevation fields may be `null` during bootstrap. Do not guess them. Add them only when a source has been checked.

Generated variants:

- `full`: all records
- `local`: current US West subset, derived by region rule

Both variants are generated from the same canonical JSON.

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

## When the Snowpool Schema Changes

Treat Snowpool's database schema and this dataset repo as separate contracts:

- The app repo owns the Supabase/Postgres table shape through migrations.
- Snowfield owns the canonical catalog content, validation rules, and generated release artifacts.
- Migrations should not become the long-term owner of canonical snow field rows.

When `snow_fields` changes in Snowpool:

1. Add the Supabase migration in the app repo.
2. Decide whether the new column is canonical catalog data or app/runtime metadata.
3. If it is canonical catalog data, add the field once in `schema/snow_fields.schema.json`.
   The schema's `required` list controls required fields; its `x-snowfield` metadata controls CSV export fields, client JSON fields, and local variant region rules.
4. Add values for the new field in `data/snow_fields.json`.
5. Update this README when the public data contract changes.
6. Update the Go CLI only when the field needs custom validation, transform, or export behavior that is not covered by the schema metadata.
7. Bump `schema_version` when the dataset release contract changes in a breaking way.
8. Regenerate and verify artifacts:

```sh
make validate
make export
```
