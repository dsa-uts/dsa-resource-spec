package resource_test

import (
	"archive/zip"
	"bytes"
	"errors"
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
	return copyFixture(t, "testdata/resource/valid/basic")
}

// copyFixture is only for unit tests that mutate files in memory.
func copyFixture(t *testing.T, dir string) fstest.MapFS {
	t.Helper()
	m := fstest.MapFS{}
	err := fs.WalkDir(os.DirFS(dir), ".", func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !e.IsDir() {
			b, err := os.ReadFile(filepath.Join(dir, p))
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

func TestJobVisibility(t *testing.T) {
	for _, producer := range []string{"", "public", "private"} {
		for _, consumer := range []string{"", "public", "private"} {
			t.Run("producer="+producer+"/consumer="+consumer, func(t *testing.T) {
				m := fixture(t)
				// The first two visibility declarations belong to build and public.
				parts := strings.SplitN(string(m["resource.yaml"].Data), "        visibility: public\n", 3)
				if len(parts) != 3 {
					t.Fatal("fixture must declare two public jobs")
				}
				data := parts[0]
				for i, visibility := range []string{producer, consumer} {
					if visibility != "" {
						data += "        visibility: " + visibility + "\n"
					}
					data += parts[i+1]
				}
				m["resource.yaml"].Data = []byte(data)
				r, err := resource.Read(m)
				if producer == "private" && consumer != "private" {
					if err == nil || !strings.Contains(err.Error(), "public Job depends on private Job") {
						t.Fatalf("expected public/private dependency rejection, got %v", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				jobs := r.Definition.Workflows["main"].Jobs
				for id, want := range map[string]string{"build": producer, "public": consumer, "private": "private"} {
					if want == "" {
						want = "public"
					}
					if got := jobs[id].Visibility; got != want {
						t.Errorf("job %s visibility = %q, want %q", id, got, want)
					}
				}
			})
		}
	}
}

// Each fixture is a complete filesystem; never overlay it onto another case.
func testFixtures(t *testing.T, kind string, validate func(fs.FS) error) {
	t.Helper()
	for _, outcome := range []string{"valid", "invalid"} {
		base := filepath.Join("testdata", kind, outcome)
		entries, err := os.ReadDir(base)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) == 0 {
			t.Fatalf("no fixtures in %s", base)
		}
		for _, entry := range entries {
			t.Run(outcome+"/"+entry.Name(), func(t *testing.T) {
				if !entry.IsDir() {
					t.Fatal("fixture must be a directory")
				}
				err := validate(os.DirFS(filepath.Join(base, entry.Name())))
				if outcome == "valid" && err != nil {
					t.Fatal(err)
				}
				if outcome == "invalid" && err == nil {
					t.Fatal("invalid fixture accepted")
				}
			})
		}
	}
}

func TestFixtures(t *testing.T) {
	testFixtures(t, "resource", resource.Validate)
}

func TestDefinitionRejections(t *testing.T) {
	for name, replacement := range map[string]struct{ before, after string }{
		"unknown field":      {"resource:", "unknown: true\nresource:"},
		"working-directory":  {"      build:\n", "      build:\n        working-directory: /workspace\n"},
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

func TestReadHardlink(t *testing.T) {
	// Git does not preserve hardlinks, so this case needs a temporary filesystem.
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS("testdata/resource/valid/basic")); err != nil {
		t.Fatal(err)
	}
	description := filepath.Join(dir, "description.md")
	if err := os.Remove(description); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(dir, "expected.txt"), description); err != nil {
		t.Fatal(err)
	}
	r, err := resource.Read(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(r.Files["description.md"], r.Files["expected.txt"]) {
		t.Fatal("hardlinked materials differ")
	}
}

func TestSymlinkFixtures(t *testing.T) {
	for _, name := range []string{"symlink", "directory-symlink", "definition-symlink"} {
		t.Run(name, func(t *testing.T) {
			err := resource.Validate(os.DirFS(filepath.Join("testdata/resource/invalid", name)))
			if !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("expected missing file after filtering symlink, got %v", err)
			}
		})
	}
}

func TestReadIgnoresUnusedSymlinks(t *testing.T) {
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS("testdata/resource/valid/basic")); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{
		"file-link": "description.md", "directory-link": ".",
		"broken-link": "missing", "external-link": t.TempDir(), "cycle": "cycle",
	} {
		if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := resource.Read(os.DirFS(dir)); err != nil {
		t.Fatal(err)
	}
}

func TestReadZIP(t *testing.T) {
	for _, linked := range []string{"unused", "description.md", "resource.yaml", "material"} {
		t.Run(linked, func(t *testing.T) {
			m := fixture(t)
			if linked == "material" {
				m["resource.yaml"].Data = bytes.ReplaceAll(m["resource.yaml"].Data, []byte("description.md"), []byte("material/description.md"))
			}
			m[linked] = &fstest.MapFile{Data: []byte("description.md"), Mode: fs.ModeSymlink | 0777}
			var archive bytes.Buffer
			writer := zip.NewWriter(&archive)
			for name, file := range m {
				header := &zip.FileHeader{Name: name, Method: zip.Deflate}
				header.SetMode(file.Mode)
				out, err := writer.CreateHeader(header)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := out.Write(file.Data); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			root, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
			if err != nil {
				t.Fatal(err)
			}
			r, err := resource.Read(root)
			if linked != "unused" {
				if !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("expected missing file, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(r.Files["description.md"], m["description.md"].Data) {
				t.Fatal("ZIP material differs from source")
			}
		})
	}
}

// unreadableFS exposes directory entries but fails when opening one file.
type unreadableFS struct {
	fs.FS
	name string
}

func (root unreadableFS) Open(name string) (fs.File, error) {
	if name == root.name {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return root.FS.Open(name)
}

func TestReadIncludesUnreferencedFilesInMemory(t *testing.T) {
	m := fixture(t)
	m["unused.txt"] = &fstest.MapFile{Data: []byte("unused")}
	if _, err := resource.Read(unreadableFS{FS: m, name: "unused.txt"}); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("expected file read failure, got %v", err)
	}
	r, err := resource.Read(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Files["unused.txt"]; ok {
		t.Fatal("unreferenced file returned as material")
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

func TestReadExclusions(t *testing.T) {
	for _, name := range []string{".env", ".git/config", ".github/workflows/check.yml", "node_modules/pkg/index.js", "nested/.hidden/file", "nested/node_modules/pkg/index.js"} {
		t.Run(name, func(t *testing.T) {
			m := fixture(t)
			m[name] = &fstest.MapFile{Data: []byte("excluded")}
			if _, err := resource.Read(unreadableFS{FS: m, name: name}); err != nil {
				t.Fatalf("excluded entry was read: %v", err)
			}
			m["resource.yaml"].Data = bytes.ReplaceAll(m["resource.yaml"].Data, []byte("description.md"), []byte(name))
			if _, err := resource.Read(m); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("expected excluded material to be missing, got %v", err)
			}
		})
	}
}

func TestReadNodeModulesFile(t *testing.T) {
	m := fixture(t)
	m["node_modules"] = m["description.md"]
	m["resource.yaml"].Data = bytes.ReplaceAll(m["resource.yaml"].Data, []byte("description.md"), []byte("node_modules"))
	if _, err := resource.Read(m); err != nil {
		t.Fatal(err)
	}
}

func TestReadRejectsSpecialFiles(t *testing.T) {
	for _, mode := range []fs.FileMode{fs.ModeNamedPipe, fs.ModeSocket, fs.ModeDevice} {
		m := fixture(t)
		m["special"] = &fstest.MapFile{Mode: mode}
		if _, err := resource.Read(m); err == nil || !strings.Contains(err.Error(), "not a regular file") {
			t.Fatalf("expected special file rejection for %v, got %v", mode, err)
		}
	}
}

func TestReadRejectsDirectoryAsFile(t *testing.T) {
	for _, name := range []string{"resource.yaml", "description.md", "expected.txt"} {
		t.Run(name, func(t *testing.T) {
			m := fixture(t)
			m[name] = &fstest.MapFile{Mode: fs.ModeDir | 0755}
			if _, err := resource.Read(m); err == nil || !strings.Contains(err.Error(), "not a regular file") {
				t.Fatalf("expected directory rejection, got %v", err)
			}
		})
	}
}
