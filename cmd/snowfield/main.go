package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/superfunllc/snowfield/internal/snowfield"
)

const defaultDatasetPath = "data/snow_fields.json"
const defaultSchemaPath = "schema/snow_fields.schema.json"
const defaultOutputDir = "dist"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "validate":
		err = runValidate(os.Args[2:])
	case "export":
		err = runExport(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: snowfield <command> [options]

commands:
  validate   validate the canonical dataset
  export     generate CSV, GeoJSON, client JSON, and manifest artifacts`)
}

func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	datasetPath := fs.String("dataset", defaultDatasetPath, "path to dataset JSON")
	schemaPath := fs.String("schema", defaultSchemaPath, "path to field catalog JSON schema")
	if err := fs.Parse(args); err != nil {
		return err
	}

	loaded, err := snowfield.Load(*datasetPath, *schemaPath)
	if err != nil {
		return err
	}
	if errors := snowfield.Validate(loaded); len(errors) > 0 {
		fmt.Fprintln(os.Stderr, "dataset validation failed:")
		for _, validationError := range errors {
			fmt.Fprintf(os.Stderr, "- %s\n", validationError)
		}
		return fmt.Errorf("validation failed")
	}

	fmt.Printf("validated %d snow field records from %s\n", len(loaded.Dataset.Records), *datasetPath)
	return nil
}

func runExport(args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	datasetPath := fs.String("dataset", defaultDatasetPath, "path to dataset JSON")
	schemaPath := fs.String("schema", defaultSchemaPath, "path to field catalog JSON schema")
	outputDir := fs.String("output-dir", defaultOutputDir, "directory for generated artifacts")
	generatedAt := fs.String("generated-at", "", "RFC3339 timestamp for generated artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	loaded, err := snowfield.Load(*datasetPath, *schemaPath)
	if err != nil {
		return err
	}
	if errors := snowfield.Validate(loaded); len(errors) > 0 {
		fmt.Fprintln(os.Stderr, "dataset validation failed:")
		for _, validationError := range errors {
			fmt.Fprintf(os.Stderr, "- %s\n", validationError)
		}
		return fmt.Errorf("validation failed")
	}

	paths, err := snowfield.Export(loaded, *outputDir, *generatedAt)
	if err != nil {
		return err
	}
	for _, path := range paths {
		fmt.Println(path)
	}
	return nil
}

