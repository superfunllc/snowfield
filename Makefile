SNOWFIELD ?= go run ./cmd/snowfield
OUTPUT_DIR ?= dist
DATASET ?= data/snow_fields.json
SCHEMA ?= schema/snow_fields.schema.json

.PHONY: validate export sync-legacy sync-catalog sync-catalog-dry-run test clean

validate:
	$(SNOWFIELD) validate --dataset $(DATASET) --schema $(SCHEMA)

export:
	$(SNOWFIELD) export --dataset $(DATASET) --schema $(SCHEMA) --output-dir $(OUTPUT_DIR)

sync-legacy:
	$(SNOWFIELD) sync --dataset $(DATASET) --schema $(SCHEMA) --schema-mode legacy

sync-catalog:
	$(SNOWFIELD) sync --dataset $(DATASET) --schema $(SCHEMA) --schema-mode catalog

sync-catalog-dry-run:
	$(SNOWFIELD) sync --dataset $(DATASET) --schema $(SCHEMA) --schema-mode catalog --variant full --dry-run

test:
	go test ./...

clean:
	rm -rf $(OUTPUT_DIR)
