package resource

import (
	"fmt"
	"io/fs"
	"path"
	"reflect"
	"strings"
)

func relative(p string) error {
	if p == "." || !fs.ValidPath(p) || strings.ContainsAny(p, "\\:\x00") {
		return fmt.Errorf("invalid relative path %q", p)
	}
	return nil
}

// readRegular checks each directory entry before opening it. fs.FS adapters must
// faithfully expose entry types (and link counts when available), and remain stable.
func readRegular(root fs.FS, name string) ([]byte, error) {
	if err := relative(name); err != nil {
		return nil, err
	}
	parent := "."
	parts := strings.Split(name, "/")
	for i, part := range parts {
		entries, err := fs.ReadDir(root, parent)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		var entry fs.DirEntry
		for _, e := range entries {
			if e.Name() == part {
				entry = e
				break
			}
		}
		if entry == nil {
			return nil, fmt.Errorf("%s: %w", name, fs.ErrNotExist)
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return nil, fmt.Errorf("%s: symlink is forbidden", name)
		}
		if i < len(parts)-1 {
			if !info.IsDir() {
				return nil, fmt.Errorf("%s: not a directory", name)
			}
		} else {
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("%s: not a regular file", name)
			}
			if hasMultipleLinks(info) {
				return nil, fmt.Errorf("%s: hardlink is forbidden", name)
			}
		}
		parent = path.Join(parent, part)
	}
	return fs.ReadFile(root, name)
}

func hasMultipleLinks(info fs.FileInfo) bool {
	v := reflect.ValueOf(info.Sys())
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.IsValid() && v.Kind() == reflect.Struct {
		n := v.FieldByName("Nlink")
		if n.IsValid() {
			if n.CanUint() {
				return n.Uint() > 1
			}
			if n.CanInt() {
				return n.Int() > 1
			}
		}
	}
	return false
}
