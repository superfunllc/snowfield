#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path
from typing import Any
from urllib import error, parse, request

from export import rows_for_variant
from field_catalog import sync_columns, sync_conflict_column
from validate import load_dataset, validate_dataset


def payload_for_mode(rows: list[dict[str, Any]], schema_mode: str) -> tuple[list[dict[str, Any]], str]:
    columns = sync_columns(schema_mode)
    conflict_column = sync_conflict_column(schema_mode)
    return [{column: row.get(column) for column in columns} for row in rows], conflict_column


def upsert_rows(
    supabase_url: str,
    service_role_key: str,
    table: str,
    rows: list[dict[str, Any]],
    conflict_column: str,
    dry_run: bool,
) -> None:
    if dry_run:
        print(json.dumps({"table": table, "on_conflict": conflict_column, "rows": rows}, indent=2))
        return

    endpoint = (
        f"{supabase_url.rstrip('/')}/rest/v1/{parse.quote(table)}"
        f"?on_conflict={parse.quote(conflict_column)}"
    )
    body = json.dumps(rows).encode("utf-8")
    req = request.Request(endpoint, data=body, method="POST")
    req.add_header("apikey", service_role_key)
    req.add_header("Authorization", f"Bearer {service_role_key}")
    req.add_header("Content-Type", "application/json")
    req.add_header("Prefer", "resolution=merge-duplicates,return=minimal")

    try:
        with request.urlopen(req, timeout=30) as resp:
            if resp.status < 200 or resp.status >= 300:
                raise RuntimeError(f"supabase status {resp.status}: {resp.read().decode('utf-8')}")
    except error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"supabase status {exc.code}: {detail}") from exc


def main() -> int:
    parser = argparse.ArgumentParser(description="Upsert Snowfield data into Supabase via PostgREST.")
    parser.add_argument("--dataset", default="data/snow_fields.json", type=Path)
    parser.add_argument("--variant", choices=["full", "local"], default="full")
    parser.add_argument("--schema-mode", choices=["legacy", "catalog"], default="legacy")
    parser.add_argument("--table", default=os.environ.get("SUPABASE_SNOW_FIELDS_TABLE", "snow_fields"))
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    try:
        dataset = load_dataset(args.dataset)
        errors = validate_dataset(dataset)
        if errors:
            print("dataset validation failed:", file=sys.stderr)
            for error_message in errors:
                print(f"- {error_message}", file=sys.stderr)
            return 1
        rows = rows_for_variant(dataset["records"], args.variant)
        payload, conflict_column = payload_for_mode(rows, args.schema_mode)

        supabase_url = os.environ.get("SUPABASE_URL")
        service_role_key = os.environ.get("SUPABASE_SERVICE_ROLE_KEY")
        if not args.dry_run and (not supabase_url or not service_role_key):
            raise RuntimeError("SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY are required unless --dry-run is set")

        upsert_rows(
            supabase_url=supabase_url or "https://example.supabase.co",
            service_role_key=service_role_key or "dry-run",
            table=args.table,
            rows=payload,
            conflict_column=conflict_column,
            dry_run=args.dry_run,
        )
    except Exception as exc:
        print(f"sync failed: {exc}", file=sys.stderr)
        return 1

    print(f"upserted {len(payload)} {args.variant} rows into {args.table} using {args.schema_mode} mode")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
