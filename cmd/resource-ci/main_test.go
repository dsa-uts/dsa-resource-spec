package main

import (
	"os"
	"os/exec"
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
		{[]string{"check"}, "requires --base"},
		{[]string{"check", "--base", "HEAD", "extra"}, "unexpected arguments"},
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
	for _, args := range [][]string{
		{"init", "-b", "main"}, {"add", "."},
		{"-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "initial"},
	} {
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git: %v: %s", err, output)
		}
	}
	if err := run([]string{"check", "--root", root, "--base", "HEAD"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample/expected.txt"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"check", "--root", root, "--base", "HEAD"}); err == nil || !strings.Contains(err.Error(), "bump resource.version") {
		t.Fatalf("expected version check failure, got %v", err)
	}
	// build-images shares validation and accepts repositories with no image builds.
	if err := run([]string{"build-images", "--root", root}); err != nil {
		t.Fatal(err)
	}
}
