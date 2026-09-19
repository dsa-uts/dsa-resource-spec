package resource

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/distribution/reference"
	"github.com/santhosh-tekuri/jsonschema/v5"
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
	Build Build `yaml:"build" json:"build"`
}

type Build struct {
	Context    string   `yaml:"context" json:"context"`
	Dockerfile string   `yaml:"dockerfile" json:"dockerfile"`
	Image      string   `yaml:"image" json:"image"`
	Platforms  []string `yaml:"platforms" json:"platforms"`
}

//go:embed schemas/resources.schema.json
var manifestSchemaBytes []byte
var manifestSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	if err := c.AddResource("manifest.json", bytes.NewReader(manifestSchemaBytes)); err != nil {
		return nil, err
	}
	return c.Compile("manifest.json")
})

var pinnedBase = regexp.MustCompile(`^FROM [a-z0-9./_-]+:[A-Za-z0-9_.-]+@sha256:[a-f0-9]{64}$`)

// ReadManifest validates the author index, resources and local image build inputs.
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
	if err := validateBuilds(root, manifest.SandboxImages); err != nil {
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

func validateBuilds(root fs.FS, images map[string]ImageBuild) error {
	repositories := map[string]bool{}
	for id, image := range images {
		build := image.Build
		if err := relative(build.Context); err != nil {
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
		if err := validateBuildContext(root, build.Context); err != nil {
			return fmt.Errorf("image %s: %w", id, err)
		}
		if err := validateDockerfile(root, build.Dockerfile); err != nil {
			return fmt.Errorf("image %s: %w", id, err)
		}
	}
	return nil
}

func validateBuildContext(root fs.FS, context string) error {
	// Check directory components without following symlinks, including empty contexts.
	parent := "."
	for _, part := range strings.Split(context, "/") {
		entries, err := fs.ReadDir(root, parent)
		if err != nil {
			return err
		}
		found := false
		for _, entry := range entries {
			if entry.Name() == part {
				found = entry.IsDir() && entry.Type()&fs.ModeSymlink == 0
			}
		}
		if !found {
			return fmt.Errorf("invalid build context: %s", context)
		}
		parent = path.Join(parent, part)
	}
	err := fs.WalkDir(root, context, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Name() == ".git" || entry.Name() == "resource.yaml" || entry.Name() == "resources.yaml" {
			return fmt.Errorf("build context must exclude .git and manifests")
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("build symlink forbidden: %s", name)
		}
		if !entry.IsDir() {
			_, err = readRegular(root, name)
		}
		return err
	})
	if err != nil {
		return err
	}
	return nil
}

func validateDockerfile(root fs.FS, name string) error {
	dockerfile, err := readRegular(root, name)
	if err != nil {
		return err
	}
	bases := 0
	for _, line := range strings.Split(string(dockerfile), "\n") {
		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)
		if strings.HasPrefix(strings.ToLower(line), "# syntax=") {
			return fmt.Errorf("external Dockerfile frontends unsupported")
		}
		if strings.HasPrefix(upper, "FROM ") {
			bases++
			if !pinnedBase.MatchString(line) {
				return fmt.Errorf("Dockerfile requires tag+digest pinned FROM")
			}
		}
		if strings.HasPrefix(upper, "ADD ") {
			return fmt.Errorf("ADD unsupported; use COPY")
		}
	}
	if bases != 1 {
		return fmt.Errorf("Dockerfile requires one pinned FROM")
	}
	ignore := name + ".dockerignore"
	if _, err := readRegular(root, ignore); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
