# GitHub Release Import Plan

## Goal

Use GitHub Releases as the distribution point for Snowfield dataset artifacts, and have the Snowpool backend pull those artifacts into Supabase on a cadence.

This removes the need for this repo's GitHub Actions workflow to hold `SUPABASE_URL` or `SUPABASE_SERVICE_ROLE_KEY`.

## Current Release Artifacts

The dataset workflow already generates and attaches the contents of `dist/` to GitHub Releases:

- `snow_fields.full.csv`
- `snow_fields.full.geojson`
- `snow_fields.full.client.json`
- `snow_fields.full.manifest.json`
- `snow_fields.local.csv`
- `snow_fields.local.geojson`
- `snow_fields.local.client.json`
- `snow_fields.local.manifest.json`

The backend should import from the client JSON artifact and use the matching manifest as its control file.

For the full dataset variant:

```text
https://github.com/superfunllc/snowfield/releases/latest/download/snow_fields.full.manifest.json
https://github.com/superfunllc/snowfield/releases/latest/download/snow_fields.full.client.json
```

For the local dataset variant:

```text
https://github.com/superfunllc/snowfield/releases/latest/download/snow_fields.local.manifest.json
https://github.com/superfunllc/snowfield/releases/latest/download/snow_fields.local.client.json
```

GitHub documents this `releases/latest/download/<asset-name>` URL shape for latest release asset links:

```text
https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases
```

## Import Owner

The Snowpool backend should own the import job.

Reasons:

- The app repo owns the Supabase/Postgres table shape through migrations.
- Import behavior can stay close to the schema assumptions it depends on.
- Backend logs, retries, alerts, and deployment controls are the natural operational surface for runtime data ingestion.
- GitHub Actions only needs permission to create GitHub release artifacts and does not need any Supabase production credential.

## Backend Import Flow

1. Run on a cadence, for example hourly or daily.
2. Fetch the latest variant manifest.
3. Validate manifest basics:
   - `dataset_name` is `snow_fields`
   - `variant` matches the configured import variant
   - `schema_version` is supported by the backend
   - `dataset_version` is present and well-formed
   - `assets.client_json.path` matches the expected artifact name
   - `assets.client_json.sha256` is present
   - `assets.client_json.row_count` is present
4. Compare `dataset_version` to the last successful import. If already imported, exit without writing.
5. Fetch the matching `snow_fields.<variant>.client.json` artifact.
6. Compute SHA-256 and compare it with `manifest.assets.client_json.sha256`.
7. Parse and validate the dataset payload.
8. Upsert rows into `snow_fields` in a transaction.
9. Record the import result, including:
   - `dataset_version`
   - `schema_version`
   - `variant`
   - imported row count
   - artifact SHA-256
   - started/completed timestamps
   - release URL or resolved release tag
   - success/failure status and error message

## Database Access

Prefer a limited backend database role for the importer if the backend architecture allows it.

The role should only have the privileges needed to import the dataset, such as:

- `USAGE` on the target schema
- `SELECT`, `INSERT`, and `UPDATE` on `snow_fields`
- access to the import audit table, if one is added

Avoid storing the Supabase `service_role` key in this dataset repo.

## Latest vs Pinned Releases

Using `releases/latest/download/...` is simple and works well for a scheduled puller.

For a more deterministic import, the backend can resolve the latest release first through the GitHub Releases API, record the release tag, then download assets from that tag:

```text
https://github.com/superfunllc/snowfield/releases/download/<tag>/snow_fields.full.manifest.json
https://github.com/superfunllc/snowfield/releases/download/<tag>/snow_fields.full.client.json
```

That gives better auditability because the import record can point at the exact release tag used.

GitHub's release API can list assets and exposes `browser_download_url` for each asset:

```text
https://docs.github.com/en/rest/releases/releases
https://docs.github.com/en/rest/releases/assets
```

## Security Notes

This model removes direct production database credentials from GitHub Actions, but it does not make GitHub Releases a trusted oracle by itself.

The manifest checksum protects against download corruption or mismatched artifacts. It does not protect against a malicious release created by someone with repository release permissions, because that actor can publish both a modified artifact and a matching manifest.

Mitigations:

- Protect the release branch/tag process.
- Require review for dataset changes.
- Keep release creation limited to trusted maintainers or protected automation.
- Have the backend allow only supported `schema_version` values.
- Keep imports idempotent and transactional.
- Store import history so rollback and audit are possible.

## Future Hardening

Potential improvements:

- Add `git_sha` and release tag metadata to each generated manifest.
- Add a top-level aggregate manifest for all variants.
- Add an importer dry-run mode in the backend.
- Add alerting for failed imports or schema-version mismatches.
- Remove the manual Supabase sync path from this repo once the backend importer is production-ready.
