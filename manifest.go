package resource

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
)

// Manifest contains resolved resources in registration order and CI image build settings.
type Manifest struct {
	Resources     []Resource            `json:"resources"`
	SandboxImages map[string]ImageBuild `json:"sandbox-images"`
	// SourceHashes covers each definition and its referenced materials, before image tag resolution.
	SourceHashes map[string]string `json:"-"`
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
	if err := validateBuilds(root, images); err != nil {
		return nil, err
	}
	manifest := &Manifest{Resources: make([]Resource, 0, len(input.Resources)), SandboxImages: images, SourceHashes: make(map[string]string)}
	ids, paths := map[string]bool{}, map[string]bool{}
	for _, entry := range input.Resources {
		if ids[entry.ID] || paths[entry.Path] {
			return nil, fmt.Errorf("duplicate resource ID/path")
		}
		ids[entry.ID], paths[entry.Path] = true, true
		if err := validateSourcePath(entry.Path); err != nil {
			return nil, err
		}
		reader := &sourceReader{root: root, hashes: make(map[string]string)}
		resource, err := loadResource(reader, entry.Path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Path, err)
		}
		if resource.Metadata.ID != entry.ID {
			return nil, fmt.Errorf("resource ID mismatch: %s", entry.ID)
		}
		manifest.Resources = append(manifest.Resources, *resource)
		manifest.SourceHashes[entry.ID] = reader.hash()
	}
	return manifest, nil
}

func validateBuilds(root *os.Root, images map[string]ImageBuild) error {
	for id, build := range images {
		if err := validateSourcePath(build.Context); err != nil {
			return fmt.Errorf("sandbox-images.%s.context: %w", id, err)
		}
		if err := validateSourcePath(build.Dockerfile); err != nil {
			return fmt.Errorf("sandbox-images.%s.dockerfile: %w", id, err)
		}
		context, err := root.Stat(build.Context)
		if err != nil {
			return fmt.Errorf("sandbox-images.%s.context: %w", id, err)
		}
		if !context.IsDir() {
			return fmt.Errorf("sandbox-images.%s.context: %q: not a directory", id, build.Context)
		}
		dockerfile, err := root.Stat(build.Dockerfile)
		if err != nil {
			return fmt.Errorf("sandbox-images.%s.dockerfile: %w", id, err)
		}
		if !dockerfile.Mode().IsRegular() {
			return fmt.Errorf("sandbox-images.%s.dockerfile: %q: not a regular file", id, build.Dockerfile)
		}
	}
	return nil
}
