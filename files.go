package resource

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// validateRuntimePath requires a non-empty POSIX relative path without
// dot, dot-dot, or empty components.
func validateRuntimePath(p string) error {
	if p == "." || !fs.ValidPath(p) || strings.ContainsAny(p, "\\:\x00") {
		return fmt.Errorf("invalid relative path %q", p)
	}
	return nil
}

// validateSourcePath allows dot and dot-dot components. The caller must resolve
// the path through os.Root to enforce containment without lexical cleaning.
func validateSourcePath(name string) error {
	if name == "" || strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\:\x00") {
		return fmt.Errorf("invalid source path %q", name)
	}
	return nil
}

func readMaterial(root *os.Root, name string) ([]byte, bool, error) {
	// Leave .. and symlink resolution to os.Root. Cleaning the path here would
	// change the meaning of a symlink followed by .. .
	if err := validateSourcePath(name); err != nil {
		return nil, false, err
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
