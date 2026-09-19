package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// decodeYAML checks the document shape before decoding into the requested type.
func decodeYAML(data []byte, schema *jsonschema.Schema, destination any) error {
	var node yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&node); err != nil {
		return err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("expected one YAML document")
	}
	if err := checkNode(&node); err != nil {
		return err
	}
	var value any
	if err := node.Decode(&value); err != nil {
		return err
	}
	// JSON Schema expects JSON numeric types, not YAML's Go integer values.
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

func checkNode(n *yaml.Node) error {
	if n.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Tag != "!!str" || seen[k.Value] {
				return fmt.Errorf("duplicate or non-string YAML key at line %d", k.Line)
			}
			seen[k.Value] = true
		}
	}
	for _, child := range n.Content {
		if err := checkNode(child); err != nil {
			return err
		}
	}
	return nil
}
