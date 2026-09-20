package resource

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// relative validates runtime destinations, not authoring references.
func relative(p string) error {
	if p == "." || !fs.ValidPath(p) || strings.ContainsAny(p, "\\:\x00") {
		return fmt.Errorf("invalid relative path %q", p)
	}
	return nil
}

func readMaterial(root *os.Root, name string) ([]byte, bool, error) {
	// Leave .. and symlink resolution to os.Root. Cleaning the path here would
	// change the meaning of a symlink followed by .. .
	if name == "" || strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\:\x00") {
		return nil, false, fmt.Errorf("invalid source path %q", name)
	}
	info, err := root.Stat(name)
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf("%s: not a regular file", name)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	info, err = file.Stat()
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf("%s: not a regular file", name)
	}
	content, err := io.ReadAll(file)
	return content, info.Mode().Perm()&0111 != 0, err
}

func sourceMaterial(root *os.Root, dir, name string) ([]byte, bool, error) {
	if name == "" || strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\:\x00") {
		return nil, false, fmt.Errorf("invalid source path %q", name)
	}
	return readMaterial(root, dir+"/"+name)
}
