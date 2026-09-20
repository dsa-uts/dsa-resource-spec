package main

import (
	"bytes"
	"encoding/json"
	resource "github.com/dsa-uts/dsa-resource-spec"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCommands(t *testing.T) {
	for _, command := range []string{"validate", "catalog", "show"} {
		var out bytes.Buffer
		args := []string{command, "../../testdata/resource/valid/basic"}
		if command == "show" {
			args = append(args, "sample")
		}
		if err := run(args, &out); err != nil {
			t.Fatal(err)
		}
		if command == "show" {
			if _, err := resource.DecodeResource(&out); err != nil {
				t.Fatal(err)
			}
		}
		if command == "catalog" {
			var got any
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			var want any
			if err := json.Unmarshal([]byte(`{"resources":[{"id":"sample","name":"Test resource","version":"v1.0.0"}],"sandbox-images":{}}`), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("catalog = %s", out.Bytes())
			}
		}
		if command == "validate" && out.Len() != 0 {
			t.Fatal("validate emitted JSON")
		}
	}
	for _, args := range [][]string{nil, {"unknown"}, {"show", "../../testdata/resource/valid/basic"}, {"show", "../../testdata/resource/valid/basic", "missing"}, {"validate", ".", "extra"}} {
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestCLIFixtures(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "resource-spec")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	for _, kind := range []string{"resource", "manifest"} {
		commands := []string{"validate", "catalog", "show"}
		if kind == "manifest" {
			commands = []string{"validate", "catalog"}
		}
		for _, outcome := range []string{"valid", "invalid"} {
			base := filepath.Join("../../testdata", kind, outcome)
			entries, err := os.ReadDir(base)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				for _, command := range commands {
					t.Run(command+"/"+outcome+"/"+entry.Name(), func(t *testing.T) {
						args := []string{command, filepath.Join(base, entry.Name())}
						if command == "show" {
							args = append(args, "sample")
						}
						output, err := exec.Command(binary, args...).CombinedOutput()
						if outcome == "valid" && err != nil {
							t.Fatalf("%v\n%s", err, output)
						}
						if outcome == "invalid" {
							if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
								t.Fatalf("expected exit code 1, got %v\n%s", err, output)
							}
							if len(output) == 0 {
								t.Fatal("missing validation error")
							}
						}
					})
				}
			}
		}
	}
}
