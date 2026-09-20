package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/jsonschema-go/jsonschema"
	"gopkg.in/yaml.v3"
)

// decodeYAML checks the document shape before decoding into the requested type.
func decodeYAML(data []byte, schema *jsonschema.Resolved, destination any) error {
	var node yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&node); err != nil {
		return err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("expected one YAML document")
	}
	var value any
	if err := node.Decode(&value); err != nil {
		return err
	}
	// Normalize YAML values to JSON before validation.
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(encoded, &value); err != nil {
		return err
	}
	if err := schema.Validate(value); err != nil {
		return err
	}
	return node.Decode(destination)
}
