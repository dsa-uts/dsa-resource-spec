package main

import (
	"bytes"
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
