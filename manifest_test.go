package resource_test

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	resource "github.com/dsa-uts/dsa-resource-spec"
)

func manifestFixture(t *testing.T) fstest.MapFS {
	t.Helper()
	m := fstest.MapFS{}
	for p, f := range fixture(t) {
		m["sample/"+p] = f
	}
	m["resources.yaml"] = &fstest.MapFile{Data: []byte("resources:\n- id: sample\n  path: sample/resource.yaml\nsandbox-images: {}\n")}
	return m
}

func TestManifest(t *testing.T) {
	if _, err := resource.ReadManifest(manifestFixture(t)); err != nil {
		t.Fatal(err)
	}
}

func TestManifestRejections(t *testing.T) {
	for name, replacement := range map[string]struct{ before, after string }{
		"id mismatch":        {"id: sample", "id: other"},
		"old history":        {"path: sample/resource.yaml", "path: sample/resource.yaml\n  versions: []"},
		"duplicate key":      {"resources:", "resources: []\nresources:"},
		"path escape":        {"sample/resource.yaml", "../sample/resource.yaml"},
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
	for _, name := range []string{"id-mismatch", "duplicate-yaml-key"} {
		t.Run(name, func(t *testing.T) {
			m := manifestFixture(t)
			file := "sample/resource.yaml"
			if name == "duplicate-yaml-key" {
				file = "resources.yaml"
			}
			b, err := os.ReadFile("testdata/invalid/" + name + "/" + file)
			if err != nil {
				t.Fatal(err)
			}
			m[file] = &fstest.MapFile{Data: b}
			if _, err := resource.ReadManifest(m); err == nil {
				t.Fatal("invalid fixture accepted")
			}
		})
	}
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
