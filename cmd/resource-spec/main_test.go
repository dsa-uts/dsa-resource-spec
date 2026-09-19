package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCommands(t *testing.T) {
	var out bytes.Buffer
	for _, args := range [][]string{{"validate", "../../testdata/resource/valid/basic"}, {"inspect", "../../testdata/resource/valid/basic"}, {"manifest", "../../testdata/manifest/valid/basic"}, {"compare", "v1.10.0", "v1.2.0"}} {
		if err := run(args, &out); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCLIFixtures(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "resource-spec")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	for _, kind := range []string{"resource", "manifest"} {
		commands := []string{"validate", "inspect"}
		if kind == "manifest" {
			commands = []string{"manifest"}
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
						output, err := exec.Command(binary, command, filepath.Join(base, entry.Name())).CombinedOutput()
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
