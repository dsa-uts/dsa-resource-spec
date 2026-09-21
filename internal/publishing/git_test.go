package publishing

import (
	"errors"
	"strings"
	"testing"
)

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
