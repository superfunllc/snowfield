#!/usr/bin/env python3
from __future__ import annotations

import argparse
import csv
import hashlib
import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from field_catalog import fields_with_flag, local_regions
from validate import load_dataset, validate_dataset


CSV_FIELDS = fields_with_flag("csv")
MIN_JSON_FIELDS = fields_with_flag("min_json")
LOCAL_REGIONS = local_regions()


def generated_at_now() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def rows_for_variant(records: list[dict[str, Any]], variant: str) -> list[dict[str, Any]]:
    if variant == "full":
        return records
    if variant == "local":
        return [
            record
            for record in records
            if (record["country_code"], record["region_code"]) in LOCAL_REGIONS
        ]
    raise ValueError(f"unknown variant {variant!r}")


def csv_value(value: Any) -> str:
    if value is None:
        return ""
    if isinstance(value, bool):
        return "true" if value else "false"
    return str(value)


def write_csv(path: Path, rows: list[dict[str, Any]]) -> None:
    with path.open("w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=CSV_FIELDS, extrasaction="ignore")
        writer.writeheader()
        for row in rows:
            writer.writerow({field: csv_value(row.get(field)) for field in CSV_FIELDS})


def write_min_json(path: Path, rows: list[dict[str, Any]]) -> None:
    payload = [{field: row.get(field) for field in MIN_JSON_FIELDS} for row in rows]
    path.write_text(json.dumps(payload, indent=2, sort_keys=False) + "\n", encoding="utf-8")


def write_geojson(path: Path, rows: list[dict[str, Any]]) -> int:
    features = []
    for row in rows:
        lat = row.get("lat")
        lng = row.get("lng")
        if lat is None or lng is None:
            continue
        properties = {field: row.get(field) for field in CSV_FIELDS if field not in {"lat", "lng"}}
        features.append(
            {
                "type": "Feature",
                "id": row["catalog_id"],
                "geometry": {
                    "type": "Point",
                    "coordinates": [lng, lat],
                },
                "properties": properties,
            }
        )

    payload = {
        "type": "FeatureCollection",
        "features": features,
    }
    path.write_text(json.dumps(payload, indent=2, sort_keys=False) + "\n", encoding="utf-8")
    return len(features)


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def write_manifest(
    path: Path,
    dataset: dict[str, Any],
    variant: str,
    rows: list[dict[str, Any]],
    generated_at: str,
    csv_path: Path,
    geojson_path: Path,
    min_json_path: Path,
    geocoded_row_count: int,
) -> None:
    manifest = {
        "dataset_name": dataset["dataset_name"],
        "dataset_version": dataset["dataset_version"],
        "schema_version": dataset["schema_version"],
        "variant": variant,
        "generated_at": generated_at,
        "row_count": len(rows),
        "geocoded_row_count": geocoded_row_count,
        "sha256": sha256(csv_path),
        "assets": {
            "csv": {
                "path": csv_path.name,
                "sha256": sha256(csv_path),
                "row_count": len(rows),
            },
            "geojson": {
                "path": geojson_path.name,
                "sha256": sha256(geojson_path),
                "feature_count": geocoded_row_count,
            },
            "min_json": {
                "path": min_json_path.name,
                "sha256": sha256(min_json_path),
                "row_count": len(rows),
            },
        },
    }
    path.write_text(json.dumps(manifest, indent=2, sort_keys=False) + "\n", encoding="utf-8")


def export_dataset(dataset: dict[str, Any], output_dir: Path, generated_at: str) -> list[Path]:
    output_dir.mkdir(parents=True, exist_ok=True)
    records = dataset["records"]
    written: list[Path] = []

    for variant in ["full", "local"]:
        rows = rows_for_variant(records, variant)
        csv_path = output_dir / f"snow_fields.{variant}.csv"
        geojson_path = output_dir / f"snow_fields.{variant}.geojson"
        min_json_path = output_dir / f"snow_fields.{variant}.min.json"
        manifest_path = output_dir / f"snow_fields.{variant}.manifest.json"

        write_csv(csv_path, rows)
        write_min_json(min_json_path, rows)
        geocoded_row_count = write_geojson(geojson_path, rows)
        write_manifest(
            manifest_path,
            dataset,
            variant,
            rows,
            generated_at,
            csv_path,
            geojson_path,
            min_json_path,
            geocoded_row_count,
        )
        written.extend([csv_path, geojson_path, min_json_path, manifest_path])

    return written


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate Snowfield release artifacts.")
    parser.add_argument("--dataset", default="data/snow_fields.json", type=Path)
    parser.add_argument("--output-dir", default="dist", type=Path)
    parser.add_argument("--generated-at", default=generated_at_now())
    args = parser.parse_args()

    try:
        dataset = load_dataset(args.dataset)
    except Exception as exc:
        print(f"failed to load {args.dataset}: {exc}", file=sys.stderr)
        return 1

    errors = validate_dataset(dataset)
    if errors:
        print("dataset validation failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    written = export_dataset(dataset, args.output_dir, args.generated_at)
    for path in written:
        print(path)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
