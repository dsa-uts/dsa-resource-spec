// Package publishing implements this repository's GitHub publication workflow.
// Resource parsing and validation remain in the resource library.
package publishing

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type commandResult struct {
	stdout string
	stderr string
}

type commandRunner func(directory string, stream bool, name string, args ...string) (commandResult, error)

func execute(directory string, stream bool, name string, args ...string) (commandResult, error) {
	command := exec.Command(name, args...)
	command.Dir = directory
	if name == "git" {
		// Push retry classification below uses Git's English rejection messages.
		command.Env = append(os.Environ(), "LC_ALL=C")
	}
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if stream {
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
	}
	err := command.Run()
	result := commandResult{stdout.String(), stderr.String()}
	if err != nil {
		return result, fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, result.stderr)
	}
	return result, nil
}

type publisher struct {
	run commandRunner
}

func (p publisher) git(directory string, args ...string) (string, error) {
	result, err := p.run(directory, false, "git", args...)
	return strings.TrimSpace(result.stdout), err
}
