// Package resource loads author manifests and validates self-contained resources.
package resource

import (
	"encoding/json"
	"fmt"
	"io"
)

// Resource contains resolved data and retains neither source paths nor file handles.
// Marshal it with encoding/json to save or distribute it.
type Resource struct {
	Metadata  Metadata            `json:"metadata"`
	Workflows map[string]Workflow `json:"workflows"`
}

// DecodeResource restores one resolved JSON resource, rejecting unknown fields
// and invalid runtime data. It does not read files or apply authoring defaults.
func DecodeResource(r io.Reader) (*Resource, error) {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	var result Resource
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("expected one JSON document")
	}
	if err := validateResource(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func loadResource(root *sourceReader, dir string) (*Resource, error) {
	data, _, err := root.read(dir + "/resource.yaml")
	if err != nil {
		return nil, err
	}
	input, err := decodeDefinition(data)
	if err != nil {
		return nil, fmt.Errorf("resource.yaml: %w", err)
	}
	result := &Resource{Metadata: Metadata(input.Resource), Workflows: make(map[string]Workflow)}
	for id, raw := range input.Workflows {
		workflow, err := resolveWorkflow(root, dir, raw)
		if err != nil {
			return nil, fmt.Errorf("workflow %s: %w", id, err)
		}
		result.Workflows[id] = workflow
	}
	if err := validateResource(result); err != nil {
		return nil, err
	}
	return result, nil
}
