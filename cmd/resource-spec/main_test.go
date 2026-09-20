package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCLI(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "resource-spec")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if output, err := exec.CommandContext(ctx, "go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	execute := func(t *testing.T, success bool, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var stdout, stderr bytes.Buffer
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		if ctx.Err() != nil {
			t.Fatalf("CLI timed out: %v", args)
		}
		if success {
			if err != nil {
				t.Fatalf("%v: %v\nstderr:\n%s", args, err, &stderr)
			}
		} else {
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() <= 0 {
				t.Fatalf("%v: expected nonzero exit status, got %v\nstderr:\n%s", args, err, &stderr)
			}
		}
		return stdout.Bytes()
	}
	for _, outcome := range []string{"valid", "invalid"} {
		base := filepath.Join("../../testdata/cli", outcome)
		entries, err := os.ReadDir(base)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) == 0 {
			t.Fatalf("no fixtures in %s", base)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			t.Run(outcome+"/"+entry.Name(), func(t *testing.T) {
				dir := filepath.Join(base, entry.Name())
				input := filepath.Join(dir, "input")
				if outcome == "invalid" {
					execute(t, false, "validate", input)
					return
				}
				t.Run("validate", func(t *testing.T) {
					if got := execute(t, true, "validate", input); len(got) != 0 {
						t.Fatalf("unexpected stdout: %s", got)
					}
				})
				t.Run("catalog", func(t *testing.T) {
					compareJSON(t, execute(t, true, "catalog", input), filepath.Join(dir, "want/catalog.json"))
				})
				// Read IDs from the checked-in catalog so a missing show golden cannot silently skip a resource.
				var catalog struct {
					Resources []struct {
						ID string `json:"id"`
					} `json:"resources"`
				}
				data, err := os.ReadFile(filepath.Join(dir, "want/catalog.json"))
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(data, &catalog); err != nil {
					t.Fatal(err)
				}
				for _, item := range catalog.Resources {
					t.Run("show/"+item.ID, func(t *testing.T) {
						compareJSON(t, execute(t, true, "show", input, item.ID), filepath.Join(dir, "want/show", item.ID+".json"))
					})
				}
			})
		}
	}
	basic := "../../testdata/cli/valid/basic/input"
	invalid := "../../testdata/cli/invalid/missing-expected/input"
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no-command", nil}, {"unknown-command", []string{"unknown"}},
		{"missing-directory", []string{"validate"}},
		{"missing-id", []string{"show", basic}},
		{"unknown-id", []string{"show", basic, "missing"}},
		{"extra-argument", []string{"validate", basic, "extra"}},
		{"unknown-flag", []string{"catalog", "--unknown"}},
		{"catalog-invalid-input", []string{"catalog", invalid}},
		{"show-invalid-input", []string{"show", invalid, "sample"}},
	} {
		t.Run("arguments/"+tc.name, func(t *testing.T) { execute(t, false, tc.args...) })
	}
	// Absolute targets depend on the checkout location; prepare only these cases at runtime.
	t.Run("absolute-references", func(t *testing.T) {
		for _, kind := range []string{"material", "material-link", "resource-link", "build-context-link", "build-dockerfile-link"} {
			t.Run(kind, func(t *testing.T) {
				dir := t.TempDir()
				if err := os.CopyFS(dir, os.DirFS(basic)); err != nil {
					t.Fatal(err)
				}
				target, err := filepath.Abs(filepath.Join(basic, "sample"))
				if err != nil {
					t.Fatal(err)
				}
				file, before, after := "sample/resource.yaml", "description-path: description.md", "description-path: "+filepath.Join(target, "description.md")
				switch kind {
				case "material-link":
					if err := os.Symlink(filepath.Join(target, "description.md"), filepath.Join(dir, "sample/link")); err != nil {
						t.Fatal(err)
					}
					after = "description-path: link"
				case "resource-link":
					if err := os.Symlink(target, filepath.Join(dir, "linked")); err != nil {
						t.Fatal(err)
					}
					file, before, after = "manifest.yaml", "path: sample", "path: linked"
				case "build-context-link", "build-dockerfile-link":
					outside := t.TempDir()
					if err := os.WriteFile(filepath.Join(outside, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(outside, filepath.Join(dir, "external")); err != nil {
						t.Fatal(err)
					}
					context, dockerfile := ".", "external/Dockerfile"
					if kind == "build-context-link" {
						context = "external"
					}
					data := "resources: []\nsandbox-images:\n  default:\n    context: " + context + "\n    dockerfile: " + dockerfile + "\n    image: registry.example.com/example/default:v1\n    platforms: [linux/amd64]\n"
					if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(data), 0644); err != nil {
						t.Fatal(err)
					}
					execute(t, false, "validate", dir)
					return
				}
				path := filepath.Join(dir, file)
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(strings.ReplaceAll(string(data), before, after)), 0644); err != nil {
					t.Fatal(err)
				}
				execute(t, false, "validate", dir)
			})
		}
	})
}

func compareJSON(t *testing.T, got []byte, path string) {
	t.Helper()
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decode := func(data []byte) any {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			t.Fatalf("%s: invalid JSON: %v\n%s", path, err, data)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			t.Fatalf("%s: expected exactly one JSON document", path)
		}
		return value
	}
	actual, expected := decode(got), decode(want)
	if !reflect.DeepEqual(actual, expected) {
		formatted, _ := json.MarshalIndent(actual, "", "  ")
		t.Fatalf("JSON differs from %s\nwant:\n%s\ngot:\n%s", path, want, formatted)
	}
}
