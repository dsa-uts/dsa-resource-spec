package resource_test

import (
	"bytes"
	"encoding/json"
	resource "github.com/dsa-uts/dsa-resource-spec"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDecodeRejections(t *testing.T) {
	original, err := os.ReadFile("testdata/cli/valid/basic/want/show/sample.json")
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(map[string]any){
		"empty required file":      func(v map[string]any) { v["required-files"] = []string{""} },
		"blank required file":      func(v map[string]any) { v["required-files"] = []string{" \t\n\u3000"} },
		"non-string required file": func(v map[string]any) { v["required-files"] = []any{1} },
		"old definition":           func(v map[string]any) { v["definition"] = map[string]any{} },
		"unknown source":           func(v map[string]any) { jsonWorkflow(v)["source"] = "secret" },
		"empty workflows":          func(v map[string]any) { v["workflows"] = nil },
		"invalid version":          func(v map[string]any) { v["metadata"].(map[string]any)["version"] = "v1" },
		"dependency":               func(v map[string]any) { jsonJob(v)["depends"] = []any{"missing"} },
		"duplicate dependency":     func(v map[string]any) { jsonJob(v)["depends"] = []any{"build", "build"} },
		"visibility":               func(v map[string]any) { jsonJob(v)["visibility"] = "secret" },
		"timeout":                  func(v map[string]any) { jsonStep(v)["timeout"] = 0 },
		"negative timeout":         func(v map[string]any) { jsonStep(v)["timeout"] = -1 },
		"timeout overflow":         func(v map[string]any) { jsonStep(v)["timeout"] = 1e30 },
		"memory":                   func(v map[string]any) { jsonJob(v)["limits"].(map[string]any)["memory"] = -1 },
		"missing limits":           func(v map[string]any) { delete(jsonJob(v), "limits") },
		"zero CPU":                 func(v map[string]any) { jsonJob(v)["limits"].(map[string]any)["cpu"] = 0 },
		"negative CPU":             func(v map[string]any) { jsonJob(v)["limits"].(map[string]any)["cpu"] = -1 },
		"fractional CPU":           func(v map[string]any) { jsonJob(v)["limits"].(map[string]any)["cpu"] = 1.5 },
		"exit code":                func(v map[string]any) { jsonStep(v)["expected"].(map[string]any)["exit-code"] = 256 },
		"negative exit code":       func(v map[string]any) { jsonStep(v)["expected"].(map[string]any)["exit-code"] = -1 },
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

func TestRequiredFiles(t *testing.T) {
	for _, tc := range []struct {
		name, field, want string
		invalid           bool
	}{
		{"omitted", "", `[]`, false},
		{"empty", "required-files: []\n", `[]`, false},
		{"display text", "required-files: [main.c, '*.c', ' レポート.pdf（任意） ', main.c]\n", `["main.c","*.c"," レポート.pdf（任意） ","main.c"]`, false},
		{"empty string", "required-files: ['']\n", "", true},
		{"blank string", "required-files: ['   ']\n", "", true},
		{"unicode blank", "required-files: ['\u3000']\n", "", true},
		{"number", "required-files: [123]\n", "", true},
		{"scalar", "required-files: main.c\n", "", true},
		{"null", "required-files: null\n", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS("testdata/cli/valid/basic/input")); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "sample/resource.yaml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append([]byte(tc.field), data...), 0644); err != nil {
				t.Fatal(err)
			}
			manifest, err := resource.LoadManifest(dir)
			if tc.invalid {
				if err == nil {
					t.Fatal("invalid required-files accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err = json.Marshal(manifest.Resources[0])
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			if got := string(fields["required-files"]); got != tc.want {
				t.Fatalf("required-files = %s, want %s", got, tc.want)
			}
			decoded, err := resource.DecodeResource(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decoded.RequiredFiles, manifest.Resources[0].RequiredFiles) {
				t.Fatal("required-files changed during JSON round trip")
			}
		})
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

// Decode the published JSON without loading its source manifest or materials.
func TestDecodeRoundTrip(t *testing.T) {
	paths, err := filepath.Glob("testdata/cli/valid/*/want/show/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no resource JSON fixtures")
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := resource.DecodeResource(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(decoded)
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			for _, item := range []struct {
				data   []byte
				target *any
			}{{encoded, &got}, {data, &want}} {
				decoder := json.NewDecoder(bytes.NewReader(item.data))
				decoder.UseNumber()
				if err := decoder.Decode(item.target); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("JSON round trip changed %s", path)
			}
			for _, legacy := range []string{
				strings.Replace(string(data), `"required-files": [],`, "", 1),
				strings.Replace(string(data), `"required-files": []`, `"required-files": null`, 1),
			} {
				restored, err := resource.DecodeResource(strings.NewReader(legacy))
				if err != nil {
					t.Fatal(err)
				}
				if restored.RequiredFiles == nil || len(restored.RequiredFiles) != 0 {
					t.Fatal("missing/null required-files was not restored as an empty array")
				}
			}
		})
	}
}
