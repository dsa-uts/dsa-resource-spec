package resource

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/spf13/afero"
)

func relative(p string) error {
	if p == "." || !fs.ValidPath(p) || strings.ContainsAny(p, "\\:\x00") {
		return fmt.Errorf("invalid relative path %q", p)
	}
	return nil
}

// readFiles loads regular files, excluding symbolic links, dot names and node_modules directories.
// The caller must supply stable entries with accurate file types.
func readFiles(root fs.FS) (afero.Fs, error) {
	files := afero.NewMemMapFs()
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
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			return files.MkdirAll(name, 0755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("%s: not a regular file", name)
		}
		data, err := fs.ReadFile(root, name)
		if err != nil {
			return err
		}
		return afero.WriteFile(files, name, data, 0644)
	})
	return files, err
}

func readFile(files afero.Fs, name string) ([]byte, error) {
	if err := relative(name); err != nil {
		return nil, err
	}
	info, err := files.Stat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: not a regular file", name)
	}
	return afero.ReadFile(files, name)
}
