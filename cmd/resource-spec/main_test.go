package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCommands(t *testing.T) {
	var out bytes.Buffer
	for _, args := range [][]string{{"validate", "../../testdata/valid"}, {"inspect", "../../testdata/valid"}, {"compare", "v1.10.0", "v1.2.0"}} {
		if err := run(args, &out); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFailedArchiveDoesNotReplaceOutput(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "existing.zip")
	if err := os.WriteFile(dest, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"archive", "--output", dest, "../../testdata/valid"}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected unresolved image error")
	}
	b, err := os.ReadFile(dest)
	if err != nil || string(b) != "keep" {
		t.Fatal("existing output changed", err)
	}
}

func TestArchiveOutputInsideResourceRejected(t *testing.T) {
	if err := run([]string{"archive", "--output", "../../testdata/valid/out.zip", "../../testdata/valid"}, &bytes.Buffer{}); err == nil {
		t.Fatal("accepted output inside resource")
	}
}
