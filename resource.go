// Package resource validates and eagerly reads locally available resource definitions.
package resource

import (
	"fmt"
	"io/fs"
)

// Resource holds the validated definition and every referenced file, keyed by its resource-relative path.
// It does not retain the input filesystem or perform subsequent I/O.
type Resource struct {
	Definition Definition
	Files      map[string][]byte
}

// Validate performs the same definition and referenced-file checks as Read.
func Validate(root fs.FS) error {
	_, err := Read(root)
	return err
}

// Read validates a resource and reads all referenced materials before returning.
// The caller must supply a stable filesystem throughout the call.
func Read(root fs.FS) (*Resource, error) {
	data, err := readRegular(root, "resource.yaml")
	if err != nil {
		return nil, err
	}
	definition, err := decodeDefinition(data)
	if err != nil {
		return nil, fmt.Errorf("resource.yaml: %w", err)
	}
	resource := &Resource{Definition: *definition, Files: map[string][]byte{}}
	for id, workflow := range definition.Workflows {
		if err := validateWorkflow(workflow); err != nil {
			return nil, fmt.Errorf("workflow %s: %w", id, err)
		}
		if err := resource.readMaterials(root, workflow); err != nil {
			return nil, fmt.Errorf("workflow %s: %w", id, err)
		}
	}
	return resource, nil
}

func (r *Resource) readMaterials(root fs.FS, workflow Workflow) error {
	paths := []string{workflow.DescriptionPath}
	if workflow.Presets != nil {
		for _, preset := range workflow.Presets.Files {
			paths = append(paths, preset.Source)
		}
	}
	for _, job := range workflow.Jobs {
		for _, step := range job.Steps {
			streams := []*Stream{step.Stdin}
			if step.Expected != nil {
				streams = append(streams, step.Expected.Stdout, step.Expected.Stderr)
			}
			for _, stream := range streams {
				if stream != nil {
					paths = append(paths, stream.Path)
				}
			}
		}
	}
	for _, name := range paths {
		if name == "" {
			continue
		}
		if _, loaded := r.Files[name]; loaded {
			continue
		}
		data, err := readRegular(root, name)
		if err != nil {
			return err
		}
		r.Files[name] = data
	}
	return nil
}
