package resource

import (
	"bytes"
	_ "crypto/sha256"
	_ "embed"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/distribution/reference"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"golang.org/x/mod/semver"
)

//go:embed schemas/resource.schema.json
var schemaBytes []byte
var compiledSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	if err := c.AddResource("resource.json", bytes.NewReader(schemaBytes)); err != nil {
		return nil, err
	}
	return c.Compile("resource.json")
})

func decode(data []byte) (*Definition, error) {
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
	return &definition, nil
}

func relative(p string) error {
	if p == "." || !fs.ValidPath(p) || strings.ContainsAny(p, "\\:\x00") {
		return fmt.Errorf("invalid relative path %q", p)
	}
	return nil
}

func imageReference(s string) (reference.Named, error) {
	r, err := reference.ParseNamed(s)
	if err != nil {
		return nil, fmt.Errorf("invalid fully qualified image %q: %w", s, err)
	}
	_, tag := r.(reference.Tagged)
	_, digest := r.(reference.Digested)
	if !tag && !digest {
		return nil, fmt.Errorf("image requires tag or digest: %s", s)
	}
	return r, nil
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
	definition, err := decode(data)
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
