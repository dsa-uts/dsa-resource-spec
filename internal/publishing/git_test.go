package publishing

import (
	"archive/tar"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRevisionPreservesInternalSymlinkAndExecutable(t *testing.T) {
	f := newFixture(t)
	must(t, os.Rename(filepath.Join(f.root, "sample/expected.txt"), filepath.Join(f.root, "sample/output.txt")))
	must(t, os.Symlink("output.txt", filepath.Join(f.root, "sample/expected.txt")))
	must(t, os.Chmod(filepath.Join(f.root, "sample/output.txt"), 0755))
	base := f.commit()
	must(t, f.publisher.check(f.root, base))
}

func TestRevisionRejectsEscapingArchive(t *testing.T) {
	for _, header := range []*tar.Header{
		{Name: "../outside", Typeflag: tar.TypeReg, Mode: 0644},
		{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "../outside"},
		{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "/tmp/outside"},
	} {
		t.Run(header.Name+header.Linkname, func(t *testing.T) {
			temp := t.TempDir()
			archive := filepath.Join(temp, "source.tar")
			file, err := os.Create(archive)
			must(t, err)
			writer := tar.NewWriter(file)
			must(t, writer.WriteHeader(header))
			must(t, writer.Close())
			must(t, file.Close())
			wantError(t, extractRevision(archive, filepath.Join(temp, "source")), "unsafe archive")
		})
	}
}

func TestPushFailuresAndWorktreeCleanup(t *testing.T) {
	for _, test := range []struct {
		message  string
		attempts int
		expected string
	}{
		{"remote: permission denied", 1, "permission denied"},
		{"rejected (non-fast-forward)", maxPushAttempts, "main kept advancing"},
	} {
		t.Run(test.message, func(t *testing.T) {
			f := newFixture(t)
			original := f.publisher.run
			attempts := 0
			f.publisher.run = func(directory string, stream bool, name string, args ...string) (commandResult, error) {
				if name == "git" && len(args) == 3 && args[0] == "push" && args[2] == "HEAD:refs/heads/main" {
					attempts++
					return commandResult{stderr: test.message}, errors.New(test.message)
				}
				return original(directory, stream, name, args...)
			}
			wantError(t, f.publisher.publish(f.root), test.expected)
			if attempts != test.attempts {
				t.Fatalf("got %d attempts, want %d", attempts, test.attempts)
			}
			if len(f.registry.calls) != 1 {
				t.Fatal("resolved image again on retry")
			}
			worktrees := f.git(f.root, "worktree", "list", "--porcelain")
			if strings.Count(worktrees, "worktree ") != 1 {
				t.Fatalf("worktree leaked: %s", worktrees)
			}
			if f.git(f.remote, "rev-parse", "main") != f.initial {
				t.Fatal("failed push changed remote")
			}
		})
	}
}
