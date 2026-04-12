#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import re
import sys
from datetime import date
from pathlib import Path
from typing import Any


DATASET_VERSION_RE = re.compile(r"^\d{4}\.\d{2}\.\d{2}([-.+][A-Za-z0-9_.-]+)?$")
CATALOG_ID_RE = re.compile(r"^[a-z0-9_.:-]+$")
SLUG_RE = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")
SOURCE_RE = re.compile(r"^[a-z0-9_]+$")
COUNTRY_RE = re.compile(r"^[A-Z]{2}$")
TAG_RE = re.compile(r"^[a-z0-9_]+$")

REQUIRED_RECORD_FIELDS = [
    "catalog_id",
    "slug",
    "source",
    "source_id",
    "name",
    "country_code",
    "region_code",
    "region_name",
    "locality",
    "timezone",
    "lat",
    "lng",
    "elevation_ft",
    "base_elevation_ft",
    "summit_elevation_ft",
    "vertical_drop_ft",
    "status",
    "is_active",
    "is_verified",
    "tags",
    "updated_at",
    "sources",
]

STATUS_ACTIVE_VALUE = {
    "active": True,
    "proposed": False,
    "inactive": False,
    "retired": False,
}


def load_dataset(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as f:
        dataset = json.load(f)
    if not isinstance(dataset, dict):
        raise ValueError("dataset root must be an object")
    return dataset


def validate_date(value: Any, path: str, errors: list[str]) -> None:
    if not isinstance(value, str):
        errors.append(f"{path}: expected YYYY-MM-DD string")
        return
    try:
        date.fromisoformat(value)
    except ValueError:
        errors.append(f"{path}: invalid date {value!r}; expected YYYY-MM-DD")


def validate_number_or_null(
    value: Any,
    path: str,
    min_value: float,
    max_value: float,
    errors: list[str],
    integer: bool = False,
) -> None:
    if value is None:
        return
    if integer:
        valid_type = isinstance(value, int) and not isinstance(value, bool)
    else:
        valid_type = isinstance(value, (int, float)) and not isinstance(value, bool)
    if not valid_type:
        expected = "integer" if integer else "number"
        errors.append(f"{path}: expected {expected} or null")
        return
    if value < min_value or value > max_value:
        errors.append(f"{path}: {value!r} outside expected range {min_value}..{max_value}")


def validate_sources(record: dict[str, Any], index: int, errors: list[str]) -> None:
    sources = record.get("sources")
    path = f"records[{index}].sources"
    if not isinstance(sources, list) or not sources:
        errors.append(f"{path}: expected non-empty list")
        return

    for source_index, source in enumerate(sources):
        source_path = f"{path}[{source_index}]"
        if not isinstance(source, dict):
            errors.append(f"{source_path}: expected object")
            continue
        for field in ["type", "name", "url", "retrieved_at"]:
            if field not in source:
                errors.append(f"{source_path}.{field}: missing required field")
        if not isinstance(source.get("type"), str) or not source.get("type"):
            errors.append(f"{source_path}.type: expected non-empty string")
        if not isinstance(source.get("name"), str) or not source.get("name").strip():
            errors.append(f"{source_path}.name: expected non-empty string")
        url = source.get("url")
        if url is not None and not isinstance(url, str):
            errors.append(f"{source_path}.url: expected string or null")
        validate_date(source.get("retrieved_at"), f"{source_path}.retrieved_at", errors)


def validate_dataset(dataset: dict[str, Any]) -> list[str]:
    errors: list[str] = []

    if dataset.get("dataset_name") != "snow_fields":
        errors.append("dataset_name: expected 'snow_fields'")

    dataset_version = dataset.get("dataset_version")
    if not isinstance(dataset_version, str) or not DATASET_VERSION_RE.match(dataset_version):
        errors.append("dataset_version: expected YYYY.MM.DD or YYYY.MM.DD-suffix")

    if dataset.get("schema_version") != 1:
        errors.append("schema_version: expected 1")

    records = dataset.get("records")
    if not isinstance(records, list):
        errors.append("records: expected list")
        return errors
    if not records:
        errors.append("records: expected at least one record")
        return errors

    catalog_ids: set[str] = set()
    slugs: set[str] = set()
    source_keys: set[tuple[str, str]] = set()
    regional_names: set[tuple[str, str, str]] = set()
    sorted_catalog_ids: list[str] = []

    for index, record in enumerate(records):
        path = f"records[{index}]"
        if not isinstance(record, dict):
            errors.append(f"{path}: expected object")
            continue

        for field in REQUIRED_RECORD_FIELDS:
            if field not in record:
                errors.append(f"{path}.{field}: missing required field")

        catalog_id = record.get("catalog_id")
        if not isinstance(catalog_id, str) or not CATALOG_ID_RE.match(catalog_id):
            errors.append(f"{path}.catalog_id: expected lowercase stable id")
        else:
            if catalog_id in catalog_ids:
                errors.append(f"{path}.catalog_id: duplicate {catalog_id!r}")
            catalog_ids.add(catalog_id)
            sorted_catalog_ids.append(catalog_id)

        slug = record.get("slug")
        if not isinstance(slug, str) or not SLUG_RE.match(slug):
            errors.append(f"{path}.slug: expected lowercase hyphenated slug")
        else:
            if slug in slugs:
                errors.append(f"{path}.slug: duplicate {slug!r}")
            slugs.add(slug)

        source = record.get("source")
        if not isinstance(source, str) or not SOURCE_RE.match(source):
            errors.append(f"{path}.source: expected lowercase source id")

        source_id = record.get("source_id")
        if not isinstance(source_id, str) or not source_id.strip():
            errors.append(f"{path}.source_id: expected non-empty string")

        if isinstance(source, str) and isinstance(source_id, str):
            source_key = (source, source_id)
            if source_key in source_keys:
                errors.append(f"{path}.source_id: duplicate source key {source_key!r}")
            source_keys.add(source_key)

        name = record.get("name")
        if not isinstance(name, str) or not name.strip():
            errors.append(f"{path}.name: expected non-empty string")

        country_code = record.get("country_code")
        if not isinstance(country_code, str) or not COUNTRY_RE.match(country_code):
            errors.append(f"{path}.country_code: expected ISO 3166-1 alpha-2 code")

        region_code = record.get("region_code")
        if not isinstance(region_code, str) or not region_code.strip():
            errors.append(f"{path}.region_code: expected non-empty string")

        if isinstance(name, str) and isinstance(country_code, str) and isinstance(region_code, str):
            regional_name = (country_code, region_code, name.casefold())
            if regional_name in regional_names:
                errors.append(f"{path}.name: duplicate name within country/region {name!r}")
            regional_names.add(regional_name)

        for string_field in ["region_name", "locality", "timezone"]:
            if not isinstance(record.get(string_field), str) or not record.get(string_field).strip():
                errors.append(f"{path}.{string_field}: expected non-empty string")

        validate_number_or_null(record.get("lat"), f"{path}.lat", -90, 90, errors)
        validate_number_or_null(record.get("lng"), f"{path}.lng", -180, 180, errors)
        if (record.get("lat") is None) != (record.get("lng") is None):
            errors.append(f"{path}: lat and lng must either both be set or both be null")

        for field in ["elevation_ft", "base_elevation_ft", "summit_elevation_ft"]:
            validate_number_or_null(record.get(field), f"{path}.{field}", -2000, 30000, errors, integer=True)
        validate_number_or_null(record.get("vertical_drop_ft"), f"{path}.vertical_drop_ft", 0, 30000, errors, integer=True)

        base = record.get("base_elevation_ft")
        summit = record.get("summit_elevation_ft")
        vertical = record.get("vertical_drop_ft")
        if isinstance(base, int) and isinstance(summit, int) and base > summit:
            errors.append(f"{path}: base_elevation_ft must be <= summit_elevation_ft")
        if isinstance(base, int) and isinstance(summit, int) and isinstance(vertical, int):
            if summit - base != vertical:
                errors.append(f"{path}: vertical_drop_ft must equal summit_elevation_ft - base_elevation_ft")

        status = record.get("status")
        if status not in STATUS_ACTIVE_VALUE:
            errors.append(f"{path}.status: expected one of {sorted(STATUS_ACTIVE_VALUE)}")
        elif record.get("is_active") is not STATUS_ACTIVE_VALUE[status]:
            errors.append(f"{path}: is_active must be {STATUS_ACTIVE_VALUE[status]} when status is {status!r}")

        if not isinstance(record.get("is_active"), bool):
            errors.append(f"{path}.is_active: expected boolean")
        if not isinstance(record.get("is_verified"), bool):
            errors.append(f"{path}.is_verified: expected boolean")

        tags = record.get("tags")
        if not isinstance(tags, list):
            errors.append(f"{path}.tags: expected list")
        else:
            seen_tags: set[str] = set()
            for tag_index, tag in enumerate(tags):
                tag_path = f"{path}.tags[{tag_index}]"
                if not isinstance(tag, str) or not TAG_RE.match(tag):
                    errors.append(f"{tag_path}: expected lowercase tag")
                    continue
                if tag in seen_tags:
                    errors.append(f"{tag_path}: duplicate tag {tag!r}")
                seen_tags.add(tag)
            if tags != sorted(tags):
                errors.append(f"{path}.tags: tags must be sorted")

        validate_date(record.get("updated_at"), f"{path}.updated_at", errors)
        validate_sources(record, index, errors)

    if sorted_catalog_ids != sorted(sorted_catalog_ids):
        errors.append("records: must be sorted by catalog_id")

    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate the snow_fields dataset.")
    parser.add_argument("--dataset", default="data/snow_fields.json", type=Path)
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

    print(f"validated {len(dataset['records'])} snow field records from {args.dataset}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

