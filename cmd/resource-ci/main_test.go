package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvalidArguments(t *testing.T) {
	for _, test := range []struct {
		args    []string
		message string
	}{
		{nil, "usage:"},
		{[]string{"unknown"}, "unknown command"},
		{[]string{"check", "--base", "HEAD"}, "flag provided but not defined"},
		{[]string{"check", "extra"}, "unexpected arguments"},
		{[]string{"publish", "--base", "HEAD"}, "flag provided but not defined"},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			err := run(test.args)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("want %q, got %v", test.message, err)
			}
		})
	}
}

func TestCheckCommand(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	root := filepath.Join(t.TempDir(), "source")
	if err := os.CopyFS(root, os.DirFS("../../testdata/cli/valid/basic/input")); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"check", "--root", root}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample/expected.txt"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"check", "--root", root}); err != nil {
		t.Fatalf("same-version source changes should be allowed: %v", err)
	}
	// build-images shares validation and accepts repositories with no image builds.
	if err := run([]string{"build-images", "--root", root}); err != nil {
		t.Fatal(err)
	}
}
