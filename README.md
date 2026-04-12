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

See `docs/DATASET_CONTRACT.md` for the field contract and release model.
