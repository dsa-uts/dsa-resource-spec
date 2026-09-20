package resource

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/distribution/reference"
	"github.com/google/jsonschema-go/jsonschema"
)

// Manifest contains resolved resources in registration order and CI image build settings.
type Manifest struct {
	Resources     []Resource            `json:"resources"`
	SandboxImages map[string]ImageBuild `json:"sandbox-images"`
}
type manifestInput struct {
	Resources     []resourceEntry          `yaml:"resources"`
	SandboxImages map[string]rawImageBuild `yaml:"sandbox-images"`
}

type resourceEntry struct {
	ID   string `yaml:"id"`
	Path string `yaml:"path"` // Resource directory relative to the manifest root.
}

//go:embed schemas/manifest.schema.json
var manifestSchemaBytes []byte
var manifestSchema = sync.OnceValues(func() (*jsonschema.Resolved, error) {
	var schema jsonschema.Schema
	if err := json.Unmarshal(manifestSchemaBytes, &schema); err != nil {
		return nil, err
	}
	return schema.Resolve(nil)
})

// LoadManifest validates manifest.yaml and resolves every registered resource.
// The caller must keep the input directory stable throughout the call.
func LoadManifest(dir string) (*Manifest, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	data, _, err := readMaterial(root, "manifest.yaml")
	if err != nil {
		return nil, err
	}
	schema, err := manifestSchema()
	if err != nil {
		return nil, err
	}
	var input manifestInput
	if err := decodeYAML(data, schema, &input); err != nil {
		return nil, fmt.Errorf("manifest.yaml: %w", err)
	}
	images := make(map[string]ImageBuild, len(input.SandboxImages))
	for id, build := range input.SandboxImages {
		images[id] = ImageBuild(build)
	}
	if err := validateBuilds(images); err != nil {
		return nil, err
	}
	manifest := &Manifest{Resources: make([]Resource, 0, len(input.Resources)), SandboxImages: images}
	ids, paths := map[string]bool{}, map[string]bool{}
	for _, entry := range input.Resources {
		if ids[entry.ID] || paths[entry.Path] {
			return nil, fmt.Errorf("duplicate resource ID/path")
		}
		ids[entry.ID], paths[entry.Path] = true, true
		if err := validateCleanRelativePath(entry.Path); err != nil {
			return nil, err
		}
		resource, err := loadResource(root, entry.Path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Path, err)
		}
		if resource.Metadata.ID != entry.ID {
			return nil, fmt.Errorf("resource ID mismatch: %s", entry.ID)
		}
		manifest.Resources = append(manifest.Resources, *resource)
	}
	return manifest, nil
}

func validateBuilds(images map[string]ImageBuild) error {
	repositories := map[string]bool{}
	for _, build := range images {
		if err := validateCleanRelativePath(build.Context); build.Context != "." && err != nil {
			return err
		}
		if err := validateCleanRelativePath(build.Dockerfile); err != nil {
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
