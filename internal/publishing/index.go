package publishing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	resource "github.com/dsa-uts/dsa-resource-spec"
	"golang.org/x/mod/semver"
)

var (
	identifier    = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	fullVersion   = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	commitPattern = regexp.MustCompile(`^[0-9a-f]{40,64}$`)
)

type releaseEntry struct {
	Path         string `json:"path"`
	SourceCommit string `json:"source-commit"`
	SourceHash   string `json:"source-hash"`
}

type releaseIndex struct {
	Resources map[string]map[string]releaseEntry `json:"resources"`
}

func releasePath(id, version string) (string, error) {
	if !identifier.MatchString(id) || !fullVersion.MatchString(version) || !semver.IsValid(version) {
		return "", fmt.Errorf("invalid resource ID/version: %s/%s", id, version)
	}
	return "release/" + id + "/" + version + ".json", nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func readIndex(root string) (releaseIndex, error) {
	index := releaseIndex{Resources: make(map[string]map[string]releaseEntry)}
	releaseDir := filepath.Join(root, "release")
	if info, err := os.Lstat(releaseDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return index, fmt.Errorf("release paths must not be symlinks")
	}
	path := filepath.Join(releaseDir, "index.json")
	data, err := readRegularFile(path)
	if os.IsNotExist(err) {
		err = checkReleaseFiles(releaseDir, map[string]bool{})
		return index, err
	}
	if err != nil {
		return index, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	index.Resources = nil
	if err := decoder.Decode(&index); err != nil {
		return index, fmt.Errorf("invalid release index: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return index, fmt.Errorf("invalid trailing release index data")
	}
	if index.Resources == nil {
		return index, fmt.Errorf("invalid release index: resources must be an object")
	}
	expected := map[string]bool{path: true}
	for id, versions := range index.Resources {
		if versions == nil {
			return index, fmt.Errorf("invalid release versions: %s", id)
		}
		for version, entry := range versions {
			relative, err := releasePath(id, version)
			if err != nil {
				return index, err
			}
			if entry.Path != relative || !commitPattern.MatchString(entry.SourceCommit) || !digestPattern.MatchString(entry.SourceHash) {
				return index, fmt.Errorf("invalid release entry: %s/%s", id, version)
			}
			file := filepath.Join(root, relative)
			if err := validateReleaseFile(file, id, version); err != nil {
				return index, err
			}
			expected[file] = true
		}
	}
	return index, checkReleaseFiles(releaseDir, expected)
}

func validateReleaseFile(path, id, version string) error {
	if info, err := os.Lstat(filepath.Dir(path)); err != nil || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("missing or unsafe release directory: %s", path)
	}
	data, err := readRegularFile(path)
	if err != nil {
		return err
	}
	item, err := resource.DecodeResource(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if item.Metadata.ID != id || item.Metadata.Version != version {
		return fmt.Errorf("release metadata mismatch: %s", path)
	}
	if err := requirePinnedImages(*item); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func readRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("missing or unsafe release file: %s", path)
	}
	return os.ReadFile(path)
}

func checkReleaseFiles(directory string, expected map[string]bool) error {
	return filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if os.IsNotExist(err) && path == directory {
			return nil
		}
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".json" && !expected[path] {
			return fmt.Errorf("release index does not match release JSON files: %s", path)
		}
		return nil
	})
}
