package publishing

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// withRevision reads committed files without checking out or executing old code.
func (p publisher) withRevision(root, ref string, action func(string) error) error {
	temp, err := os.MkdirTemp("", "resource-source-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	archive := filepath.Join(temp, "source.tar")
	if _, err := p.git(root, "archive", "--format=tar", "--output="+archive, ref); err != nil {
		return err
	}
	directory := filepath.Join(temp, "source")
	if err := extractRevision(archive, directory); err != nil {
		return err
	}
	return action(directory)
}

func extractRevision(archive, directory string) error {
	if err := os.Mkdir(directory, 0755); err != nil {
		return err
	}
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	reader := tar.NewReader(file)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// git archive includes the commit ID as a global PAX header.
		if header.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		if !filepath.IsLocal(header.Name) {
			return fmt.Errorf("unsafe archive path: %s", header.Name)
		}
		if err := root.MkdirAll(filepath.Dir(header.Name), 0755); err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := root.MkdirAll(header.Name, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			output, err := root.OpenFile(header.Name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(header.Mode)&0777)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(output, reader)
			closeErr := output.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		case tar.TypeSymlink:
			target := filepath.Join(filepath.Dir(header.Name), header.Linkname)
			if filepath.IsAbs(header.Linkname) || !filepath.IsLocal(target) {
				return fmt.Errorf("unsafe archive link: %s", header.Name)
			}
			if err := root.Symlink(header.Linkname, header.Name); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported archive entry: %s", header.Name)
		}
	}
}

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
