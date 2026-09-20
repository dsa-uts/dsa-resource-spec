package resource_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	resource "github.com/dsa-uts/dsa-resource-spec"
)

func manifestFixture(t *testing.T) fstest.MapFS {
	t.Helper()
	return copyFixture(t, "testdata/manifest/valid/basic")
}

func TestManifest(t *testing.T) {
	manifest, err := resource.ReadManifest(os.DirFS("testdata/manifest/valid/basic"))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Resources) != 1 || manifest.Resources[0].Path != "sample" {
		t.Fatalf("unexpected resources: %+v", manifest.Resources)
	}
}

func TestManifestReadsOnlyListedResources(t *testing.T) {
	m := manifestFixture(t)
	m["images/unused"] = &fstest.MapFile{}
	m["sample/unused-link"] = &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte("missing")}
	if _, err := resource.ReadManifest(unreadableFS{FS: m, name: "images/unused"}); err != nil {
		t.Fatal(err)
	}
}

func TestManifestNestedDirectory(t *testing.T) {
	m := manifestFixture(t)
	nested := fstest.MapFS{}
	for name, file := range m {
		if name != "resources.yaml" {
			nested["exercises/"+name] = file
		}
	}
	nested["resources.yaml"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(string(m["resources.yaml"].Data), "path: sample", "path: exercises/sample"))}
	if _, err := resource.ReadManifest(nested); err != nil {
		t.Fatal(err)
	}
}

func TestManifestSymlinkDirectory(t *testing.T) {
	for _, entryPath := range []string{"linked", "linked/sample"} {
		t.Run(entryPath, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS("testdata/manifest/valid/basic")); err != nil {
				t.Fatal(err)
			}
			target := "sample"
			if entryPath == "linked/sample" {
				target = "."
			}
			if err := os.Symlink(target, filepath.Join(dir, "linked")); err != nil {
				t.Fatal(err)
			}
			manifest := "resources:\n  - id: sample\n    path: " + entryPath + "\nsandbox-images: {}\n"
			if err := os.WriteFile(filepath.Join(dir, "resources.yaml"), []byte(manifest), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := resource.ReadManifest(os.DirFS(dir))
			if err == nil || !strings.Contains(err.Error(), "symlink is forbidden") {
				t.Fatalf("expected symlink rejection, got %v", err)
			}
		})
	}
}

func TestManifestRejections(t *testing.T) {
	for name, replacement := range map[string]struct{ before, after string }{
		"id mismatch":        {"id: sample", "id: other"},
		"old history":        {"path: sample", "path: sample\n  versions: []"},
		"duplicate key":      {"resources:", "resources: []\nresources:"},
		"file path":          {"path: sample", "path: sample/resource.yaml"},
		"root directory":     {"path: sample", "path: ."},
		"missing directory":  {"path: sample", "path: missing"},
		"path escape":        {"path: sample", "path: ../sample"},
		"multiple documents": {"resources:", "---\n{}\n---\nresources:"},
		"unknown field":      {"resources:", "unexpected: true\nresources:"},
		"null images":        {"sandbox-images: {}", "sandbox-images: null"},
	} {
		t.Run(name, func(t *testing.T) {
			m := manifestFixture(t)
			m["resources.yaml"].Data = []byte(strings.ReplaceAll(string(m["resources.yaml"].Data), replacement.before, replacement.after))
			if _, err := resource.ReadManifest(m); err == nil {
				t.Fatal("invalid manifest accepted")
			}
		})
	}
}

func TestManifestFixtures(t *testing.T) {
	testFixtures(t, "manifest", func(root fs.FS) error {
		_, err := resource.ReadManifest(root)
		return err
	})
}

func TestBuildConfiguration(t *testing.T) {
	m := manifestFixture(t)
	m["resources.yaml"].Data = []byte("resources: []\nsandbox-images:\n  default:\n    context: sandbox\n    dockerfile: sandbox/Dockerfile\n    image: ghcr.io/example/default\n    platforms: [linux/amd64]\n")
	manifest, err := resource.ReadManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	build := manifest.SandboxImages["default"]
	if build.Context != "sandbox" || build.Dockerfile != "sandbox/Dockerfile" ||
		build.Image != "ghcr.io/example/default" || strings.Join(build.Platforms, ",") != "linux/amd64" {
		t.Fatalf("unexpected build configuration: %+v", build)
	}
}
