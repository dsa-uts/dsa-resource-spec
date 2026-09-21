package publishing

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	resource "github.com/dsa-uts/dsa-resource-spec"
)

const maxPushAttempts = 5

var errMainAdvanced = errors.New("main advanced")

// Publish appends immutable snapshots to main, preserving concurrent source commits.
func Publish(root string) error { return (publisher{execute}).publish(root) }

func (p publisher) publish(root string) error {
	sourceCommit, err := p.git(root, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	status, err := p.git(root, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("publish requires an unchanged source checkout")
	}
	manifest, err := resource.LoadManifest(root)
	if err != nil {
		return err
	}
	// These caches outlive each worktree: push retries must reuse the exact images.
	resolved := make(map[string]resource.Resource)
	digests := make(map[string]string)
	for attempt := 1; attempt <= maxPushAttempts; attempt++ {
		err := p.withPublicationTree(root, func(tree string) error {
			index, err := readIndex(tree)
			if err != nil {
				return err
			}
			if err := p.resolveUnpublished(manifest, index, resolved, digests); err != nil {
				return err
			}
			added, err := appendReleases(tree, manifest, index, resolved, sourceCommit)
			if err != nil {
				return err
			}
			if len(added) == 0 {
				fmt.Println("All resource versions are already published.")
				return nil
			}
			if err := p.commitAndPush(tree, added); err != nil {
				return err
			}
			fmt.Printf("Published %s from %s\n", strings.Join(added, ", "), sourceCommit)
			return nil
		})
		if !errors.Is(err, errMainAdvanced) {
			return err
		}
		fmt.Printf("main advanced; retrying publication (%d/%d)\n", attempt, maxPushAttempts)
	}
	return fmt.Errorf("main kept advancing; rerun this workflow to publish")
}

func (p publisher) resolveUnpublished(manifest *resource.Manifest, index releaseIndex, resolved map[string]resource.Resource, digests map[string]string) error {
	for _, item := range manifest.Resources {
		id, version := item.Metadata.ID, item.Metadata.Version
		if existing, ok := index.Resources[id][version]; ok {
			if existing.SourceHash != manifest.SourceHashes[id] {
				return fmt.Errorf("%s/%s: published version has different source", id, version)
			}
			continue
		}
		if _, ok := resolved[id]; !ok {
			pinned, err := p.pinImages(item, digests)
			if err != nil {
				return err
			}
			resolved[id] = pinned
		}
	}
	return nil
}

func appendReleases(tree string, manifest *resource.Manifest, index releaseIndex, resolved map[string]resource.Resource, sourceCommit string) ([]string, error) {
	var added []string
	for _, item := range manifest.Resources {
		id, version := item.Metadata.ID, item.Metadata.Version
		if _, exists := index.Resources[id][version]; exists {
			continue
		}
		relative, err := releasePath(id, version)
		if err != nil {
			return nil, err
		}
		target := filepath.Join(tree, relative)
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			return nil, fmt.Errorf("refusing to overwrite %s", relative)
		}
		if info, err := os.Lstat(filepath.Dir(target)); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("refusing to write through symlink: %s", relative)
		}
		pinned, ok := resolved[id]
		if !ok {
			return nil, fmt.Errorf("missing resolved resource: %s", id)
		}
		if err := writeJSON(target, pinned); err != nil {
			return nil, err
		}
		if index.Resources[id] == nil {
			index.Resources[id] = make(map[string]releaseEntry)
		}
		index.Resources[id][version] = releaseEntry{relative, sourceCommit, manifest.SourceHashes[id]}
		added = append(added, id+"/"+version)
	}
	if len(added) > 0 {
		if err := writeJSON(filepath.Join(tree, "release/index.json"), index); err != nil {
			return nil, err
		}
		if _, err := readIndex(tree); err != nil {
			return nil, err
		}
	}
	return added, nil
}

func (p publisher) commitAndPush(tree string, added []string) error {
	if _, err := p.git(tree, "add", "--", "release"); err != nil {
		return err
	}
	if _, err := p.git(tree, "-c", "user.name=github-actions[bot]", "-c",
		"user.email=41898282+github-actions[bot]@users.noreply.github.com",
		"commit", "-m", "Publish resources: "+strings.Join(added, ", ")); err != nil {
		return err
	}
	result, err := p.run(tree, false, "git", "push", "origin", "HEAD:refs/heads/main")
	if err == nil {
		return nil
	}
	// Only a non-fast-forward race is retryable; policy/authentication errors stop here.
	if strings.Contains(result.stderr, "fetch first") || strings.Contains(result.stderr, "non-fast-forward") {
		return errMainAdvanced
	}
	return err
}
