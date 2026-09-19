package resource

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
	"golang.org/x/mod/semver"
)

type Definition struct {
	Resource  Metadata            `yaml:"resource" json:"resource"`
	Workflows map[string]Workflow `yaml:"workflows" json:"workflows"`
}

type Metadata struct {
	ID      string `yaml:"id" json:"id"`
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
}

//go:embed schemas/resource.schema.json
var schemaBytes []byte
var compiledSchema = sync.OnceValues(func() (*jsonschema.Resolved, error) {
	var schema jsonschema.Schema
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return nil, err
	}
	return schema.Resolve(nil)
})

func decodeDefinition(data []byte) (*Definition, error) {
	schema, err := compiledSchema()
	if err != nil {
		return nil, err
	}
	var definition Definition
	if err := decodeYAML(data, schema, &definition); err != nil {
		return nil, err
	}
	if !semver.IsValid(definition.Resource.Version) {
		return nil, fmt.Errorf("invalid resource.version: %q", definition.Resource.Version)
	}
	for _, workflow := range definition.Workflows {
		applyWorkflowDefaults(workflow)
	}
	return &definition, nil
}
