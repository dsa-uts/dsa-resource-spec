package resource

import (
	_ "embed"
	"encoding/json"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
)

type definition struct {
	Resource      rawMetadata            `yaml:"resource"`
	RequiredFiles []string               `yaml:"required-files"`
	Workflows     map[string]rawWorkflow `yaml:"workflows"`
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

func decodeDefinition(data []byte) (*definition, error) {
	schema, err := compiledSchema()
	if err != nil {
		return nil, err
	}
	var definition definition
	if err := decodeYAML(data, schema, &definition); err != nil {
		return nil, err
	}
	return &definition, nil
}
