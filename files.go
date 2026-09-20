package resource

import (
	"fmt"
	"io/fs"
	"reflect"
	"strings"
)

func relative(p string) error {
	if p == "." || !fs.ValidPath(p) || strings.ContainsAny(p, "\\:\x00") {
		return fmt.Errorf("invalid relative path %q", p)
	}
	return nil
}

// readFiles snapshots the resource tree without following symbolic links.
// The caller must supply stable entries with accurate file types and link counts.
func readFiles(root fs.FS) (resourceFiles, error) {
	files := resourceFiles{}
	err := fs.WalkDir(root, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s: not a regular file", name)
		}
		if hasMultipleLinks(info) {
			return fmt.Errorf("%s: hardlink is forbidden", name)
		}
		data, err := fs.ReadFile(root, name)
		if err != nil {
			return err
		}
		files[name] = data
		return nil
	})
	return files, err
}

type resourceFiles map[string][]byte

func (files resourceFiles) read(name string) ([]byte, error) {
	if err := relative(name); err != nil {
		return nil, err
	}
	data, ok := files[name]
	if !ok {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
	}
	return data, nil
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
