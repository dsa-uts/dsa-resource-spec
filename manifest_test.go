package resource_test

import (
	"os"
	"path/filepath"
	"testing"

	resource "github.com/dsa-uts/dsa-resource-spec"
)

func TestManifestOrder(t *testing.T) {
	dir := fixture(t)
	if err := os.CopyFS(filepath.Join(dir, "other"), os.DirFS(filepath.Join(dir, "sample"))); err != nil {
		t.Fatal(err)
	}
	replace(t, dir, "other/resource.yaml", "id: sample", "id: other")
	write(t, dir, "manifest.yaml", []byte("resources:\n  - id: other\n    path: other\n  - id: sample\n    path: sample\nsandbox-images: {}\n"))
	manifest, err := resource.LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Resources) != 2 || manifest.Resources[0].Metadata.ID != "other" || manifest.Resources[1].Metadata.ID != "sample" {
		t.Fatal(manifest)
	}
}

func TestManifestRejections(t *testing.T) {
	for name, pair := range map[string][2]string{
		"id mismatch":            {"id: sample", "id: other"},
		"duplicate registration": {"resources:", "resources:\n  - id: sample\n    path: sample"},
		"root directory":         {"path: sample", "path: ."},
		"missing directory":      {"path: sample", "path: missing"},
		"escape":                 {"path: sample", "path: ../sample"},
		"unknown field":          {"resources:", "unknown: true\nresources:"},
		"null images":            {"sandbox-images: {}", "sandbox-images: null"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := fixture(t)
			replace(t, dir, "manifest.yaml", pair[0], pair[1])
			if _, err := resource.LoadManifest(dir); err == nil {
				t.Fatal("invalid manifest accepted")
			}
		})
	}
}

func TestBuildConfiguration(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "manifest.yaml", []byte("resources: []\nsandbox-images:\n  default:\n    context: .\n    dockerfile: missing/Dockerfile\n    image: ghcr.io/example/default\n    platforms: [linux/amd64]\n"))
	manifest, err := resource.LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	build := manifest.SandboxImages["default"]
	if build.Context != "." || build.Dockerfile != "missing/Dockerfile" || build.Image != "ghcr.io/example/default" || len(manifest.Resources) != 0 {
		t.Fatal(manifest)
	}
}

func TestOldManifestNameRejected(t *testing.T) {
	dir := fixture(t)
	if err := os.Rename(filepath.Join(dir, "manifest.yaml"), filepath.Join(dir, "resources.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := resource.LoadManifest(dir); err == nil {
		t.Fatal("old manifest accepted")
	}
}

func TestManifestDirectorySymlink(t *testing.T) {
	for _, external := range []bool{false, true} {
		t.Run(map[bool]string{false: "internal", true: "external"}[external], func(t *testing.T) {
			dir := fixture(t)
			target := "sample"
			if external {
				outside := fixture(t)
				var err error
				target, err = filepath.Rel(dir, filepath.Join(outside, "sample"))
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(target, filepath.Join(dir, "linked")); err != nil {
				t.Fatal(err)
			}
			replace(t, dir, "manifest.yaml", "path: sample", "path: linked")
			_, err := resource.LoadManifest(dir)
			if (err != nil) != external {
				t.Fatalf("external=%v: %v", external, err)
			}
		})
	}
}
