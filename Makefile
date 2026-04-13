SNOWFIELD ?= go run ./cmd/snowfield
OUTPUT_DIR ?= dist
DATASET ?= data/snow_fields.json
SCHEMA ?= schema/snow_fields.schema.json
DATASET_VERSION ?=
GENERATED_AT ?=

EXPORT_ARGS := --dataset $(DATASET) --schema $(SCHEMA) --output-dir $(OUTPUT_DIR)
ifneq ($(strip $(DATASET_VERSION)),)
EXPORT_ARGS += --dataset-version $(DATASET_VERSION)
endif
ifneq ($(strip $(GENERATED_AT)),)
EXPORT_ARGS += --generated-at $(GENERATED_AT)
endif

.PHONY: validate export test clean

validate:
	$(SNOWFIELD) validate --dataset $(DATASET) --schema $(SCHEMA)

export:
	$(SNOWFIELD) export $(EXPORT_ARGS)


test:
	go test ./...

clean:
	rm -rf $(OUTPUT_DIR)
