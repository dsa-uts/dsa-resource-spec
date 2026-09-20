package resource_test

import (
	"os"
	"path/filepath"
	"strings"
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
	for _, tc := range []struct {
		name, context, dockerfile, invalidField string
	}{
		{"root context", ".", "images/Dockerfile", ""},
		{"nested context", "images", "images/Dockerfile", ""},
		{"dot paths", "./images", "./images/Dockerfile", ""},
		{"parent paths", "images/..", "images/../images/Dockerfile", ""},
		{"escaping context", "..", "images/Dockerfile", "context"},
		{"escaping dockerfile", ".", "../Dockerfile", "dockerfile"},
		{"absolute context", "/", "images/Dockerfile", "context"},
		{"absolute dockerfile", ".", "/Dockerfile", "dockerfile"},
		{"root dockerfile", ".", ".", "dockerfile"},
		{"symlink parent context", "nested-link/..", "images/Dockerfile", ""},
		{"symlink parent dockerfile", ".", "nested-link/../Dockerfile", ""},
		{"missing context", "missing", "images/Dockerfile", "context"},
		{"file context", "images/Dockerfile", "images/Dockerfile", "context"},
		{"missing dockerfile", ".", "missing/Dockerfile", "dockerfile"},
		{"directory dockerfile", ".", "images", "dockerfile"},
		{"internal context link", "internal", "images/Dockerfile", ""},
		{"internal dockerfile link", ".", "internal/Dockerfile", ""},
		{"external context link", "external", "images/Dockerfile", "context"},
		{"external dockerfile link", ".", "external/Dockerfile", "dockerfile"},
		{"absolute context link", "absolute", "images/Dockerfile", "context"},
		{"absolute dockerfile link", ".", "absolute/Dockerfile", "dockerfile"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, "images/Dockerfile", []byte("FROM scratch\n"))
			write(t, dir, "images/nested/placeholder", nil)
			outside := t.TempDir()
			write(t, outside, "Dockerfile", []byte("FROM scratch\n"))
			external, err := filepath.Rel(dir, outside)
			if err != nil {
				t.Fatal(err)
			}
			for name, target := range map[string]string{
				"internal":    "images",
				"nested-link": "images/nested",
				"external":    external,
				"absolute":    filepath.Join(dir, "images"),
			} {
				if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
					t.Fatal(err)
				}
			}
			write(t, dir, "manifest.yaml", []byte("resources: []\nsandbox-images:\n  default:\n    context: "+tc.context+"\n    dockerfile: "+tc.dockerfile+"\n    image: registry.example.com/example/default:v1\n    platforms: [linux/amd64]\n"))
			manifest, err := resource.LoadManifest(dir)
			if tc.invalidField != "" {
				if err == nil || !strings.Contains(err.Error(), "sandbox-images.default."+tc.invalidField) {
					t.Fatalf("expected %s error, got %v", tc.invalidField, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			build := manifest.SandboxImages["default"]
			if build.Context != tc.context || build.Dockerfile != tc.dockerfile || build.Image != "registry.example.com/example/default:v1" || len(manifest.Resources) != 0 {
				t.Fatal(manifest)
			}
		})
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
