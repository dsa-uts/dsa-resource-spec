package resource

import (
	"fmt"
	"io/fs"
	"strings"
)

func relative(p string) error {
	if p == "." || !fs.ValidPath(p) || strings.ContainsAny(p, "\\:\x00") {
		return fmt.Errorf("invalid relative path %q", p)
	}
	return nil
}

// readFiles loads regular files, excluding symbolic links, dot names and node_modules directories.
// The caller must supply stable entries with accurate file types.
func readFiles(root fs.FS) (resourceFiles, error) {
	files := resourceFiles{data: map[string][]byte{}}
	err := fs.WalkDir(root, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name != "." && (strings.HasPrefix(entry.Name(), ".") || entry.IsDir() && entry.Name() == "node_modules") {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 || entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("%s: not a regular file", name)
		}
		data, err := fs.ReadFile(root, name)
		if err != nil {
			return err
		}
		files.data[name] = data
		return nil
	})
	return files, err
}

// resourceFiles is a read-only in-memory tree. Subtrees share the same file data.
type resourceFiles struct {
	data   map[string][]byte
	prefix string
}

func (files resourceFiles) sub(name string) resourceFiles {
	return resourceFiles{data: files.data, prefix: files.prefix + name + "/"}
}

func (files resourceFiles) read(name string) ([]byte, error) {
	if err := relative(name); err != nil {
		return nil, err
	}
	data, ok := files.data[files.prefix+name]
	if !ok {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
	}
	return data, nil
}
