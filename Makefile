SNOWFIELD ?= go run ./cmd/snowfield
OUTPUT_DIR ?= dist
DATASET ?= data/snow_fields.json
SCHEMA ?= schema/snow_fields.schema.json
GENERATED_AT ?=

EXPORT_ARGS = --dataset $(DATASET) --schema $(SCHEMA) --output-dir $(OUTPUT_DIR) $(if $(strip $(GENERATED_AT)),--generated-at $(GENERATED_AT),)

.PHONY: validate export export-dev test clean

validate:
	$(SNOWFIELD) validate --dataset $(DATASET) --schema $(SCHEMA)

export:
	$(SNOWFIELD) export $(EXPORT_ARGS)

export-dev: GENERATED_AT := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
export-dev: export

test:
	go test ./...

clean:
	rm -rf $(OUTPUT_DIR)
