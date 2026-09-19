package resource

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sync"

	"github.com/distribution/reference"
	"github.com/google/jsonschema-go/jsonschema"
)

// Manifest is the author/CI index. It is not required when reading a release ZIP.
type Manifest struct {
	Resources     []ResourceEntry       `yaml:"resources" json:"resources"`
	SandboxImages map[string]ImageBuild `yaml:"sandbox-images" json:"sandbox-images"`
}

type ResourceEntry struct {
	ID   string `yaml:"id" json:"id"`
	Path string `yaml:"path" json:"path"`
}

type ImageBuild struct {
	Context    string   `yaml:"context" json:"context"`
	Dockerfile string   `yaml:"dockerfile" json:"dockerfile"`
	Image      string   `yaml:"image" json:"image"`
	Platforms  []string `yaml:"platforms" json:"platforms"`
}

//go:embed schemas/resources.schema.json
var manifestSchemaBytes []byte
var manifestSchema = sync.OnceValues(func() (*jsonschema.Resolved, error) {
	var schema jsonschema.Schema
	if err := json.Unmarshal(manifestSchemaBytes, &schema); err != nil {
		return nil, err
	}
	return schema.Resolve(nil)
})

// ReadManifest validates the author index, resources and image build configuration.
// It never builds images, contacts registries, or checks release history.
func ReadManifest(root fs.FS) (*Manifest, error) {
	data, err := readRegular(root, "resources.yaml")
	if err != nil {
		return nil, err
	}
	schema, err := manifestSchema()
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := decodeYAML(data, schema, &manifest); err != nil {
		return nil, fmt.Errorf("resources.yaml: %w", err)
	}
	if err := validateResourceEntries(root, manifest.Resources); err != nil {
		return nil, err
	}
	if err := validateBuilds(manifest.SandboxImages); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func validateResourceEntries(root fs.FS, entries []ResourceEntry) error {
	ids, paths := map[string]bool{}, map[string]bool{}
	for _, entry := range entries {
		if ids[entry.ID] || paths[entry.Path] {
			return fmt.Errorf("invalid or duplicate Resource ID/path")
		}
		ids[entry.ID] = true
		paths[entry.Path] = true
		if err := relative(entry.Path); err != nil {
			return err
		}
		if path.Base(entry.Path) != "resource.yaml" || path.Dir(entry.Path) == "." {
			return fmt.Errorf("Resource requires dedicated directory/resource.yaml")
		}
		if _, err := readRegular(root, entry.Path); err != nil {
			return err
		}
		resourceFS, err := fs.Sub(root, path.Dir(entry.Path))
		if err != nil {
			return err
		}
		resource, err := Read(resourceFS)
		if err != nil {
			return fmt.Errorf("%s: %w", entry.Path, err)
		}
		if resource.Definition.Resource.ID != entry.ID {
			return fmt.Errorf("Resource ID mismatch: %s", entry.ID)
		}
	}
	return nil
}

func validateBuilds(images map[string]ImageBuild) error {
	repositories := map[string]bool{}
	for _, build := range images {
		if err := relative(build.Context); err != nil {
			return err
		}
		if err := relative(build.Dockerfile); err != nil {
			return err
		}
		repository, err := reference.ParseNamed(build.Image)
		if err != nil || !reference.IsNameOnly(repository) || reference.Domain(repository) != "ghcr.io" {
			return fmt.Errorf("invalid GHCR build repository: %s", build.Image)
		}
		if repositories[build.Image] {
			return fmt.Errorf("duplicate build repository: %s", build.Image)
		}
		repositories[build.Image] = true
	}
	return nil
}
