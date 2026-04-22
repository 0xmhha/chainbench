// Package schema embeds the JSON Schemas that define the chainbench-net
// command/event/network contracts and exposes runtime validation helpers.
package schema

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed network.json command.json event.json
var schemaFS embed.FS

func loadSchema(name string) (*jsonschema.Schema, error) {
	path := name + ".json"
	data, err := schemaFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(path, bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("add %s: %w", path, err)
	}
	return compiler.Compile(path)
}

// ValidateBytes validates the given JSON document against the named schema
// ("network" | "command" | "event").
func ValidateBytes(name string, doc []byte) error {
	sch, err := loadSchema(name)
	if err != nil {
		return err
	}
	var v any
	if err := json.Unmarshal(doc, &v); err != nil {
		return fmt.Errorf("parse document: %w", err)
	}
	return sch.Validate(v)
}

// ValidateFile reads a JSON file and validates it against the named schema.
func ValidateFile(name, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	return ValidateBytes(name, data)
}
