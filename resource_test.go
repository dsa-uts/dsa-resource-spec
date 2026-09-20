package resource_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	resource "github.com/dsa-uts/dsa-resource-spec"
)

func write(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS("testdata/resource/valid/basic")); err != nil {
		t.Fatal(err)
	}
	return dir
}

func replace(t *testing.T, dir, name, before, after string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(before)) {
		t.Fatalf("missing replacement target %q", before)
	}
	write(t, dir, name, bytes.ReplaceAll(data, []byte(before), []byte(after)))
}

func load(t *testing.T, dir string) *resource.Resource {
	t.Helper()
	manifest, err := resource.LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	return &manifest.Resources[0]
}

func TestResolvedResource(t *testing.T) {
	dir := fixture(t)
	r := load(t, dir)
	workflow := r.Workflows["main"]
	if r.Metadata.ID != "sample" || workflow.Description == "" {
		t.Fatal(r)
	}
	job := workflow.Jobs["public"]
	if job.Limits.Memory != 64<<20 || job.Limits.CPU != 1 || job.Limits.PIDs != 128 || job.Limits.WorkspaceSize != 128<<20 || job.Limits.ArtifactSize != 1<<20 || job.Limits.StdoutSize != 10<<20 || job.Limits.StderrSize != 10<<20 {
		t.Fatal(job.Limits)
	}
	if job.Steps[0].Timeout != time.Second || job.Steps[1].Expected.ExitCode != nil || job.Steps[1].Expected.Stdout != nil {
		t.Fatal(job.Steps)
	}
	if workflow.Jobs["build"].Artifacts.Outputs[0].Visibility != "private" || !workflow.Jobs["build"].Steps[0].Compile {
		t.Fatal(workflow)
	}
	expected, err := os.ReadFile(filepath.Join(dir, "sample/expected.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(job.Steps[0].Expected.Stdout.Content, expected) {
		t.Fatal("expected output not resolved")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{`"source":`, `"description-path":`, `"definition":`, `"step-timeout":`, "expected.txt"} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatalf("source detail leaked: %s", forbidden)
		}
	}
	restored, err := resource.DecodeResource(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r, restored) {
		t.Fatal("JSON round trip changed resource")
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored.Workflows["main"].Jobs["public"].Steps[0].Expected.Stdout.Content, expected) {
		t.Fatal("resource retained filesystem")
	}
}

func TestExpectedExitCode(t *testing.T) {
	for _, value := range []string{"", "0", "1", "255", "-1", "256"} {
		t.Run("exit-code="+value, func(t *testing.T) {
			dir := fixture(t)
			if value != "" {
				replace(t, dir, "sample/resource.yaml", "          expected:\n", "          expected:\n            exit-code: "+value+"\n")
			}
			manifest, err := resource.LoadManifest(dir)
			if value == "-1" || value == "256" {
				if err == nil {
					t.Fatal("out-of-range exit code accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			r := &manifest.Resources[0]
			code := r.Workflows["main"].Jobs["public"].Steps[0].Expected.ExitCode
			encoded, err := json.Marshal(code)
			if err != nil {
				t.Fatal(err)
			}
			want := value
			if want == "" {
				want = "null"
			}
			if string(encoded) != want {
				t.Fatalf("exit code = %s, want %s", encoded, want)
			}
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(data, []byte(`"exit-code":`+want)) {
				t.Fatalf("JSON missing expected exit code: %s", data)
			}
			restored, err := resource.DecodeResource(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(r, restored) {
				t.Fatal("JSON round trip changed exit code expectations")
			}
		})
	}
}

func TestSharedMaterials(t *testing.T) {
	dir := fixture(t)
	write(t, dir, "shared/.hidden/input.bin", []byte{0, 255, 1})
	if err := os.Chmod(filepath.Join(dir, "shared/.hidden/input.bin"), 0755); err != nil {
		t.Fatal(err)
	}
	replace(t, dir, "sample/resource.yaml", "    description-path: description.md", "    description-path: ../shared/description.md\n    presets:\n      files:\n        - source: ../shared/.hidden/input.bin\n          path: tools/program")
	write(t, dir, "shared/description.md", []byte("# Shared\n"))
	replace(t, dir, "sample/resource.yaml", "- run: echo hello", "- run: echo hello\n          stdin:\n            path: ../shared/.hidden/input.bin\n          timeout: 300ms")
	replace(t, dir, "sample/resource.yaml", "path: expected.txt", "value: ''")
	r := load(t, dir)
	workflow := r.Workflows["main"]
	preset := workflow.Presets[0]
	step := workflow.Jobs["public"].Steps[0]
	if workflow.Description != "# Shared\n" || !preset.Executable || preset.Path != "tools/program" || !bytes.Equal(preset.Content, []byte{0, 255, 1}) || !bytes.Equal(step.Stdin, preset.Content) || step.Timeout != 300*time.Millisecond {
		t.Fatal(workflow)
	}
	if step.Expected.Stdout == nil || len(step.Expected.Stdout.Content) != 0 || step.Expected.Stderr != nil {
		t.Fatal(step.Expected)
	}
	data, _ := json.Marshal(r)
	restored, err := resource.DecodeResource(bytes.NewReader(data))
	if err != nil || !reflect.DeepEqual(r, restored) {
		t.Fatalf("binary/empty round trip: %v", err)
	}
}

func TestReferenceContainment(t *testing.T) {
	for _, kind := range []string{"internal-link", "external-link", "external-parent", "absolute", "directory", "unused-link", "symlink-parent"} {
		t.Run(kind, func(t *testing.T) {
			dir := fixture(t)
			outside := t.TempDir()
			write(t, outside, "secret", []byte("secret"))
			path := "description.md"
			valid := false
			switch kind {
			case "internal-link":
				if err := os.Symlink("description.md", filepath.Join(dir, "sample/link")); err != nil {
					t.Fatal(err)
				}
				path, valid = "link", true
			case "external-link":
				if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(dir, "sample/link")); err != nil {
					t.Fatal(err)
				}
				path = "link"
			case "external-parent":
				path = "../../" + filepath.Base(outside) + "/secret"
			case "absolute":
				path = filepath.Join(outside, "secret")
			case "directory":
				path = "."
			case "unused-link":
				if err := os.Symlink("missing", filepath.Join(dir, "sample/unused")); err != nil {
					t.Fatal(err)
				}
				valid = true
			case "symlink-parent":
				write(t, dir, "shared/nested/unused", nil)
				write(t, dir, "shared/description.md", []byte("shared"))
				if err := os.Symlink("../shared/nested", filepath.Join(dir, "sample/link")); err != nil {
					t.Fatal(err)
				}
				path, valid = "link/../description.md", true
			}
			replace(t, dir, "sample/resource.yaml", "description-path: description.md", "description-path: "+path)
			r, err := resource.LoadManifest(dir)
			if (err == nil) != valid {
				t.Fatalf("valid=%v, error=%v", valid, err)
			}
			if kind == "symlink-parent" && r.Resources[0].Workflows["main"].Description != "shared" {
				t.Fatal("symlink path cleaned before resolution")
			}
		})
	}
}

func TestDefinitionRejections(t *testing.T) {
	for name, pair := range map[string][2]string{
		"unknown field":        {"resource:", "unknown: true\nresource:"},
		"duplicate key":        {"resource:", "resource: {}\nresource:"},
		"multiple documents":   {"resource:", "---\n{}\n---\nresource:"},
		"unqualified image":    {"ghcr.io/example/default:latest", "default:latest"},
		"untagged image":       {"ghcr.io/example/default:latest", "ghcr.io/example/default"},
		"version":              {"v1.0.0", "v01.0.0"},
		"short version":        {"v1.0.0", "v1.0"},
		"numeric prerelease":   {"v1.0.0", "v1.0.0-01"},
		"size overflow":        {"64MiB", "9223372036854775807GiB"},
		"duration overflow":    {"1s", "9223372036854775807s"},
		"artifact destination": {"path: program", "path: ../program"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := fixture(t)
			replace(t, dir, "sample/resource.yaml", pair[0], pair[1])
			if _, err := resource.LoadManifest(dir); err == nil {
				t.Fatal("invalid definition accepted")
			}
		})
	}
}

func TestExplicitLimitsAndInlineInput(t *testing.T) {
	dir := fixture(t)
	replace(t, dir, "sample/resource.yaml", "memory: 64MiB", "memory: 1GiB\n          stdout-size: 20MiB\n          stderr-size: 21MiB\n          workspace-size: 2GiB\n          artifact-size: 2KiB\n          pids: 256")
	replace(t, dir, "sample/resource.yaml", "- run: echo hello", "- run: echo hello\n          stdin: {value: hello}")
	job := load(t, dir).Workflows["main"].Jobs["public"]
	if job.Limits.Memory != 1<<30 || job.Limits.StdoutSize != 20<<20 || job.Limits.StderrSize != 21<<20 || job.Limits.WorkspaceSize != 2<<30 || job.Limits.ArtifactSize != 2<<10 || job.Limits.PIDs != 256 || string(job.Steps[0].Stdin) != "hello" {
		t.Fatal(job)
	}
}

func TestCPULimits(t *testing.T) {
	for _, value := range []string{"1", "2", "8", "0", "-1", "1.5"} {
		t.Run(value, func(t *testing.T) {
			dir := fixture(t)
			replace(t, dir, "sample/resource.yaml", "memory: 64MiB", "memory: 64MiB\n          cpu: "+value)
			manifest, err := resource.LoadManifest(dir)
			if value == "0" || value == "-1" || value == "1.5" {
				if err == nil {
					t.Fatal("invalid CPU limit accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			r := &manifest.Resources[0]
			encodedCPU, err := json.Marshal(r.Workflows["main"].Jobs["public"].Limits.CPU)
			if err != nil || string(encodedCPU) != value {
				t.Fatalf("CPU = %s, want %s; error: %v", encodedCPU, value, err)
			}
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			restored, err := resource.DecodeResource(bytes.NewReader(data))
			if err != nil || !reflect.DeepEqual(r, restored) {
				t.Fatalf("CPU limit JSON round trip: %v", err)
			}
		})
	}
}

func TestDecodeRejections(t *testing.T) {
	original, err := json.Marshal(load(t, fixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(map[string]any){
		"old definition":       func(v map[string]any) { v["definition"] = map[string]any{} },
		"unknown source":       func(v map[string]any) { jsonWorkflow(v)["source"] = "secret" },
		"empty workflows":      func(v map[string]any) { v["workflows"] = nil },
		"invalid version":      func(v map[string]any) { v["metadata"].(map[string]any)["version"] = "v1" },
		"dependency":           func(v map[string]any) { jsonJob(v)["depends"] = []any{"missing"} },
		"duplicate dependency": func(v map[string]any) { jsonJob(v)["depends"] = []any{"build", "build"} },
		"visibility":           func(v map[string]any) { jsonJob(v)["visibility"] = "secret" },
		"timeout":              func(v map[string]any) { jsonStep(v)["timeout"] = 0 },
		"negative timeout":     func(v map[string]any) { jsonStep(v)["timeout"] = -1 },
		"timeout overflow":     func(v map[string]any) { jsonStep(v)["timeout"] = 1e30 },
		"memory":               func(v map[string]any) { jsonJob(v)["limits"].(map[string]any)["memory"] = -1 },
		"missing limits":       func(v map[string]any) { delete(jsonJob(v), "limits") },
		"zero CPU":             func(v map[string]any) { jsonJob(v)["limits"].(map[string]any)["cpu"] = 0 },
		"negative CPU":         func(v map[string]any) { jsonJob(v)["limits"].(map[string]any)["cpu"] = -1 },
		"fractional CPU":       func(v map[string]any) { jsonJob(v)["limits"].(map[string]any)["cpu"] = 1.5 },
		"exit code":            func(v map[string]any) { jsonStep(v)["expected"].(map[string]any)["exit-code"] = 256 },
		"negative exit code":   func(v map[string]any) { jsonStep(v)["expected"].(map[string]any)["exit-code"] = -1 },
		"match": func(v map[string]any) {
			jsonStep(v)["expected"].(map[string]any)["stdout"].(map[string]any)["match"] = "unknown"
		},
		"empty steps": func(v map[string]any) { jsonJob(v)["steps"] = []any{} },
	} {
		t.Run(name, func(t *testing.T) {
			var value map[string]any
			if err := json.Unmarshal(original, &value); err != nil {
				t.Fatal(err)
			}
			mutate(value)
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := resource.DecodeResource(bytes.NewReader(encoded)); err == nil {
				t.Fatal("invalid JSON accepted")
			}
		})
	}
	for _, data := range []string{string(original) + " {}", string(original) + " garbage", "null", "{}"} {
		if _, err := resource.DecodeResource(strings.NewReader(data)); err == nil {
			t.Fatal("invalid document accepted")
		}
	}
}

func jsonWorkflow(v map[string]any) map[string]any {
	return v["workflows"].(map[string]any)["main"].(map[string]any)
}
func jsonJob(v map[string]any) map[string]any {
	return jsonWorkflow(v)["jobs"].(map[string]any)["public"].(map[string]any)
}
func jsonStep(v map[string]any) map[string]any {
	return jsonJob(v)["steps"].([]any)[0].(map[string]any)
}

func TestFixtures(t *testing.T) {
	for _, kind := range []string{"resource", "manifest"} {
		for _, outcome := range []string{"valid", "invalid"} {
			base := filepath.Join("testdata", kind, outcome)
			entries, err := os.ReadDir(base)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				t.Run(kind+"/"+outcome+"/"+entry.Name(), func(t *testing.T) {
					_, err := resource.LoadManifest(filepath.Join(base, entry.Name()))
					if (err == nil) != (outcome == "valid") {
						t.Fatalf("%s: %v", outcome, err)
					}
				})
			}
		}
	}
}

func TestNestedSharedPresetSymlink(t *testing.T) {
	dir := fixture(t)
	if err := os.MkdirAll(filepath.Join(dir, "tasks"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "sample"), filepath.Join(dir, "tasks/a")); err != nil {
		t.Fatal(err)
	}
	replace(t, dir, "manifest.yaml", "path: sample", "path: tasks/a")
	write(t, dir, "shared/tool", []byte("#!/bin/sh\necho hello\n"))
	if err := os.Chmod(filepath.Join(dir, "shared/tool"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("tool", filepath.Join(dir, "shared/link")); err != nil {
		t.Fatal(err)
	}
	replace(t, dir, "tasks/a/resource.yaml", "    description-path: description.md", "    description-path: description.md\n    presets:\n      files:\n        - source: ../../shared/link\n          path: tool\n        - source: description.md\n          path: readme.md")
	presets := load(t, dir).Workflows["main"].Presets
	if !presets[0].Executable || presets[1].Executable || !bytes.HasPrefix(presets[0].Content, []byte("#!/bin/sh")) {
		t.Fatal(presets)
	}
}

func TestYAMLDefaultsWithAnchors(t *testing.T) {
	dir := fixture(t)
	data, err := os.ReadFile(filepath.Join(dir, "sample/resource.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	yaml := strings.Replace(string(data), "sandbox-image: ghcr.io/example/default:latest", "sandbox-image: &image ghcr.io/example/default:latest", 1)
	yaml = strings.ReplaceAll(yaml, "sandbox-image: ghcr.io/example/default:latest", "sandbox-image: *image")
	yaml = strings.ReplaceAll(yaml, "        visibility: public\n", "")
	write(t, dir, "sample/resource.yaml", []byte(yaml))
	jobs := load(t, dir).Workflows["main"].Jobs
	if jobs["build"].Visibility != "public" || jobs["public"].Visibility != "public" {
		t.Fatal(jobs)
	}
}
