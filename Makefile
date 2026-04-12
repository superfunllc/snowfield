PYTHON ?= python3
OUTPUT_DIR ?= dist
DATASET ?= data/snow_fields.json

.PHONY: validate export sync-legacy sync-catalog clean

validate:
	$(PYTHON) scripts/validate.py --dataset $(DATASET)

export:
	$(PYTHON) scripts/export.py --dataset $(DATASET) --output-dir $(OUTPUT_DIR)

sync-legacy:
	$(PYTHON) scripts/sync_supabase.py --dataset $(DATASET) --schema-mode legacy

sync-catalog:
	$(PYTHON) scripts/sync_supabase.py --dataset $(DATASET) --schema-mode catalog

clean:
	rm -rf $(OUTPUT_DIR)

