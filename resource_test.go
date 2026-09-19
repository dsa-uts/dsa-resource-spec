package resource_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	resource "github.com/dsa-uts/dsa-resource-spec"
)

func fixture(t *testing.T) fstest.MapFS {
	t.Helper()
	m := fstest.MapFS{}
	err := fs.WalkDir(os.DirFS("testdata/valid"), ".", func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !e.IsDir() {
			b, err := os.ReadFile(filepath.Join("testdata/valid", p))
			if err != nil {
				return err
			}
			m[p] = &fstest.MapFile{Data: b, Mode: 0644}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestRead(t *testing.T) {
	m := fixture(t)
	r, err := resource.Read(m)
	if err != nil {
		t.Fatal(err)
	}
	if r.Definition.Resource.Version != "v1.0.0" {
		t.Fatal(r.Definition.Resource)
	}
	if string(r.Files["description.md"]) == "" {
		t.Fatal("missing description")
	}
	delete(m, "description.md")
	if len(r.Files["description.md"]) == 0 {
		t.Fatal("reader retained filesystem")
	}
}

func TestFixtures(t *testing.T) {
	entries, err := os.ReadDir("testdata/invalid")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Run(e.Name(), func(t *testing.T) {
			if e.Name() == "id-mismatch" || e.Name() == "duplicate-yaml-key" {
				t.Skip("manifest-only constraint; covered by publisher validation")
			}
			m := fixture(t)
			b, err := os.ReadFile(filepath.Join("testdata/invalid", e.Name(), "sample/resource.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			m["resource.yaml"] = &fstest.MapFile{Data: b}
			if err := resource.Validate(m); err == nil {
				t.Fatal("invalid definition accepted")
			}
		})
	}
}

func TestDefinitionRejections(t *testing.T) {
	for name, replacement := range map[string]struct{ before, after string }{
		"unknown field":      {"resource:", "unknown: true\nresource:"},
		"duplicate key":      {"resource:", "resource: {}\nresource:"},
		"multiple documents": {"resource:", "---\n{}\n---\nresource:"},
		"unqualified image":  {"ghcr.io/example/default:latest", "default:latest"},
		"untagged image":     {"ghcr.io/example/default:latest", "ghcr.io/example/default"},
		"version":            {"v1.0.0", "v01.0.0"},
		"short version":      {"v1.0.0", "v1.0"},
		"numeric prerelease": {"v1.0.0", "v1.0.0-01"},
		"path escape":        {"description.md", "../description.md"},
	} {
		t.Run(name, func(t *testing.T) {
			m := fixture(t)
			m["resource.yaml"].Data = []byte(strings.ReplaceAll(string(m["resource.yaml"].Data), replacement.before, replacement.after))
			if err := resource.Validate(m); err == nil {
				t.Fatal("invalid definition accepted")
			}
		})
	}
}

func TestLinks(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink", "directory-symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			m := fixture(t)
			for p, f := range m {
				if err := os.WriteFile(filepath.Join(dir, p), f.Data, 0644); err != nil {
					t.Fatal(err)
				}
			}
			description := filepath.Join(dir, "description.md")
			if err := os.Remove(description); err != nil {
				t.Fatal(err)
			}
			var err error
			if kind == "symlink" {
				err = os.Symlink(filepath.Join(dir, "expected.txt"), description)
			} else if kind == "hardlink" {
				err = os.Link(filepath.Join(dir, "expected.txt"), description)
			} else {
				err = os.Symlink(t.TempDir(), description)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := resource.Validate(os.DirFS(dir)); err == nil {
				t.Fatal("link accepted")
			}
		})
	}
}

func TestReadAllMaterialKinds(t *testing.T) {
	m := fixture(t)
	data := string(m["resource.yaml"].Data)
	data = strings.Replace(data, "    description-path: description.md", "    description-path: description.md\n    presets:\n      files:\n      - source: material/preset.bin\n        path: include/preset.bin", 1)
	data = strings.Replace(data, "- run: echo hello", "- run: echo hello\n          stdin:\n            path: material/stdin.bin", 1)
	m["resource.yaml"].Data = []byte(data)
	m["material/preset.bin"] = &fstest.MapFile{Data: []byte{0, 255, 1}}
	m["material/stdin.bin"] = &fstest.MapFile{Data: []byte{255, 0, 2}}
	r, err := resource.Read(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"description.md", "expected.txt", "material/preset.bin", "material/stdin.bin"} {
		if !bytes.Equal(r.Files[p], m[p].Data) {
			t.Fatalf("material not loaded: %s", p)
		}
	}
	for _, p := range []string{"material/preset.bin", "material/stdin.bin"} {
		saved := m[p]
		delete(m, p)
		if err := resource.Validate(m); err == nil {
			t.Fatalf("missing material accepted: %s", p)
		}
		m[p] = saved
	}
}

func TestYAMLAnchors(t *testing.T) {
	m := fixture(t)
	data := string(m["resource.yaml"].Data)
	data = strings.Replace(data, "sandbox-image: ghcr.io/example/default:latest", "sandbox-image: &image ghcr.io/example/default:latest", 1)
	data = strings.ReplaceAll(data, "sandbox-image: ghcr.io/example/default:latest", "sandbox-image: *image")
	m["resource.yaml"].Data = []byte(data)
	if err := resource.Validate(m); err != nil {
		t.Fatal(err)
	}
}
