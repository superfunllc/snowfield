#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path
from typing import Any


DEFAULT_SCHEMA_PATH = Path(__file__).resolve().parents[1] / "schema" / "snow_fields.schema.json"


def load_field_catalog(schema_path: Path = DEFAULT_SCHEMA_PATH) -> dict[str, Any]:
    with schema_path.open("r", encoding="utf-8") as f:
        schema = json.load(f)
    if not isinstance(schema, dict):
        raise ValueError("schema root must be an object")
    return schema


FIELD_CATALOG = load_field_catalog()


def _snow_field_def(catalog: dict[str, Any] = FIELD_CATALOG) -> dict[str, Any]:
    return catalog["$defs"]["snow_field"]


def _snow_field_properties(catalog: dict[str, Any] = FIELD_CATALOG) -> dict[str, Any]:
    return _snow_field_def(catalog)["properties"]


def _source_def(catalog: dict[str, Any] = FIELD_CATALOG) -> dict[str, Any]:
    return catalog["$defs"]["source"]


def required_record_fields(catalog: dict[str, Any] = FIELD_CATALOG) -> list[str]:
    return list(_snow_field_def(catalog)["required"])


def required_source_fields(catalog: dict[str, Any] = FIELD_CATALOG) -> list[str]:
    return list(_source_def(catalog)["required"])


def fields_with_flag(flag: str, catalog: dict[str, Any] = FIELD_CATALOG) -> list[str]:
    fields: list[str] = []
    for field, field_schema in _snow_field_properties(catalog).items():
        snowfield_metadata = field_schema.get("x-snowfield", {})
        if snowfield_metadata.get(flag):
            fields.append(field)
    return fields


def sync_columns(schema_mode: str, catalog: dict[str, Any] = FIELD_CATALOG) -> list[str]:
    fields: list[str] = []
    for field, field_schema in _snow_field_properties(catalog).items():
        snowfield_metadata = field_schema.get("x-snowfield", {})
        if schema_mode in snowfield_metadata.get("sync_modes", []):
            fields.append(field)
    if not fields:
        raise ValueError(f"unknown or empty schema mode {schema_mode!r}")
    return fields


def sync_conflict_column(schema_mode: str, catalog: dict[str, Any] = FIELD_CATALOG) -> str:
    sync_modes = catalog.get("x-snowfield", {}).get("sync_modes", {})
    try:
        conflict_column = sync_modes[schema_mode]["conflict_column"]
    except KeyError as exc:
        raise ValueError(f"unknown schema mode {schema_mode!r}") from exc
    if not isinstance(conflict_column, str) or not conflict_column:
        raise ValueError(f"schema mode {schema_mode!r} must define conflict_column")
    return conflict_column


def local_regions(catalog: dict[str, Any] = FIELD_CATALOG) -> set[tuple[str, str]]:
    regions_by_country = catalog.get("x-snowfield", {}).get("local_regions", {})
    regions: set[tuple[str, str]] = set()
    for country_code, region_codes in regions_by_country.items():
        for region_code in region_codes:
            regions.add((country_code, region_code))
    return regions

