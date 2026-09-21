package publishing

import (
	"os"
	"path/filepath"
)

func (p publisher) withPublicationTree(root string, action func(string) error) (err error) {
	if _, err := p.git(root, "fetch", "origin", "main"); err != nil {
		return err
	}
	temp, err := os.MkdirTemp("", "resource-publication-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	tree := filepath.Join(temp, "main")
	if _, err := p.git(root, "worktree", "add", "--detach", tree, "origin/main"); err != nil {
		return err
	}
	defer func() {
		_, cleanupErr := p.git(root, "worktree", "remove", "--force", tree)
		if err == nil {
			err = cleanupErr
		}
	}()
	return action(tree)
}
