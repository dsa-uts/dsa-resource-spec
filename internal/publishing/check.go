package publishing

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	resource "github.com/dsa-uts/dsa-resource-spec"
	"golang.org/x/mod/semver"
)

type sourceVersion struct {
	Version string
	Hash    string
}

func sourceVersions(manifest *resource.Manifest) map[string]sourceVersion {
	sources := make(map[string]sourceVersion, len(manifest.Resources))
	for _, item := range manifest.Resources {
		sources[item.Metadata.ID] = sourceVersion{item.Metadata.Version, manifest.SourceHashes[item.Metadata.ID]}
	}
	return sources
}

// Check validates source version changes and protects previously published bytes.
func Check(root, base string) error { return (publisher{execute}).check(root, base) }

func (p publisher) check(root, base string) error {
	current, err := resource.LoadManifest(root)
	if err != nil {
		return err
	}
	index, err := readIndex(root)
	if err != nil {
		return err
	}
	ref, err := p.git(root, "rev-parse", "--verify", base+"^{commit}")
	if err != nil {
		return err
	}
	after := sourceVersions(current)
	err = p.withRevision(root, ref, func(old string) error {
		previousIndex, err := readIndex(old)
		if err != nil {
			return err
		}
		if err := checkPublishedFiles(old, root, previousIndex, index); err != nil {
			return err
		}
		previous, err := resource.LoadManifest(old)
		if err != nil {
			return err
		}
		return checkVersions(sourceVersions(previous), after, previousIndex)
	})
	if err != nil {
		return err
	}
	// Include releases that appeared after the comparison base.
	return checkVersions(nil, after, index)
}

func checkPublishedFiles(old, current string, before, after releaseIndex) error {
	for id, versions := range before.Resources {
		for version, entry := range versions {
			if after.Resources[id][version] != entry {
				return fmt.Errorf("published index entry changed: %s/%s", id, version)
			}
			previous, err := os.ReadFile(filepath.Join(old, entry.Path))
			if err != nil {
				return err
			}
			present, err := os.ReadFile(filepath.Join(current, entry.Path))
			if err != nil {
				return err
			}
			if !bytes.Equal(previous, present) {
				return fmt.Errorf("published JSON changed: %s", entry.Path)
			}
		}
	}
	return nil
}

func checkVersions(before, after map[string]sourceVersion, index releaseIndex) error {
	for id, current := range after {
		if previous, ok := before[id]; ok {
			if current.Version == previous.Version {
				if current.Hash != previous.Hash {
					return fmt.Errorf("%s: definition/materials changed; bump resource.version", id)
				}
			} else if semver.Compare(current.Version, previous.Version) <= 0 {
				return fmt.Errorf("%s: resource.version must increase", id)
			}
		}
		released := index.Resources[id]
		if existing, ok := released[current.Version]; ok {
			if existing.SourceHash != current.Hash {
				return fmt.Errorf("%s/%s: published version cannot be reused", id, current.Version)
			}
			continue
		}
		for version := range released {
			if semver.Compare(current.Version, version) <= 0 {
				return fmt.Errorf("%s: new version must exceed published versions", id)
			}
		}
	}
	return nil
}
