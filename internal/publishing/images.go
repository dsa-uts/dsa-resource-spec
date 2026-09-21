package publishing

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/distribution/reference"
	resource "github.com/dsa-uts/dsa-resource-spec"
)

var (
	ghcrRepository = regexp.MustCompile(`^ghcr\.io/[a-z0-9][a-z0-9._-]*(?:/[a-z0-9][a-z0-9._-]*)+$`)
	missingImage   = regexp.MustCompile(`(?i)(manifest unknown|name unknown|not found|\b404\b)`)
)

// BuildImages builds all registered images, publishing only changed digests.
func BuildImages(root string, ghaCache bool) error {
	return (publisher{execute}).buildImages(root, ghaCache)
}

func (p publisher) buildImages(root string, ghaCache bool) error {
	manifest, err := resource.LoadManifest(root)
	if err != nil {
		return err
	}
	names := map[string]bool{}
	for id, build := range manifest.SandboxImages {
		if !ghcrRepository.MatchString(build.Image) || names[build.Image] {
			return fmt.Errorf("%s: expected a unique, untagged GHCR repository, got %s", id, build.Image)
		}
		names[build.Image] = true
	}
	for _, id := range slices.Sorted(maps.Keys(manifest.SandboxImages)) {
		if err := p.buildImage(root, id, manifest.SandboxImages[id], ghaCache); err != nil {
			return err
		}
	}
	return nil
}

func (p publisher) buildImage(root, id string, build resource.ImageBuild, ghaCache bool) error {
	temp, err := os.MkdirTemp("", "resource-image-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	layout := filepath.Join(temp, "image")
	if _, err := p.run(root, true, "docker", buildArguments(id, build, layout, ghaCache)...); err != nil {
		return err
	}
	local, err := exportedImage(layout)
	if err != nil {
		return fmt.Errorf("%s: %w", id, err)
	}
	built, err := p.digest(local, false)
	if err != nil {
		return err
	}
	latest := build.Image + ":latest"
	previous, err := p.digest(latest, true)
	if err != nil {
		return err
	}
	if previous == built {
		fmt.Printf("%s: unchanged (%s)\n", id, built)
		return nil
	}
	timestamp := time.Now().UTC().Format("20060102150405")
	fixed := build.Image + ":" + timestamp + "-" + strings.ReplaceAll(built, ":", "-")
	// Keep a retention tag before moving latest. Verify each copy before proceeding.
	if err := p.copyImage(local, fixed, built); err != nil {
		return err
	}
	if err := p.copyImage(build.Image+"@"+built, latest, built); err != nil {
		return err
	}
	fmt.Printf("%s: published %s\n", id, fixed)
	return nil
}

func buildArguments(id string, build resource.ImageBuild, layout string, ghaCache bool) []string {
	args := []string{
		"buildx", "build", "--file", build.Dockerfile,
		"--platform", strings.Join(build.Platforms, ","),
		"--provenance=false", "--sbom=false", "--build-arg", "SOURCE_DATE_EPOCH=0",
		"--output", "type=oci,dest=" + layout + ",tar=false,rewrite-timestamp=true",
	}
	if ghaCache {
		args = append(args,
			"--cache-from", "type=gha,version=2,scope=sandbox-"+id,
			"--cache-to", "type=gha,version=2,scope=sandbox-"+id+",mode=max")
	}
	return append(args, build.Context)
}

func exportedImage(layout string) (string, error) {
	data, err := os.ReadFile(filepath.Join(layout, "index.json"))
	if err != nil {
		return "", err
	}
	var index struct {
		Manifests []struct {
			Digest string `json:"digest"`
		} `json:"manifests"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return "", err
	}
	if len(index.Manifests) != 1 || !digestPattern.MatchString(index.Manifests[0].Digest) {
		return "", fmt.Errorf("expected one exported OCI image")
	}
	return "ocidir://" + layout + "@" + index.Manifests[0].Digest, nil
}

func (p publisher) digest(image string, missingOK bool) (string, error) {
	result, err := p.run("", false, "regctl", "image", "digest", image)
	if err != nil {
		// Authentication and network errors must not be treated as absent images.
		if missingOK && missingImage.MatchString(result.stderr) {
			return "", nil
		}
		return "", fmt.Errorf("cannot resolve %s: %w", image, err)
	}
	value := strings.TrimSpace(result.stdout)
	if !digestPattern.MatchString(value) {
		return "", fmt.Errorf("invalid registry digest for %s: %s", image, value)
	}
	return value, nil
}

func (p publisher) copyImage(source, destination, expected string) error {
	if _, err := p.run("", true, "regctl", "image", "copy", source, destination); err != nil {
		return err
	}
	actual, err := p.digest(destination, false)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("digest changed while pushing %s", destination)
	}
	return nil
}

func (p publisher) pinImages(item resource.Resource, cache map[string]string) (resource.Resource, error) {
	// Workflows contain maps, so clone before updating the source snapshot.
	item.Workflows = maps.Clone(item.Workflows)
	for workflowID, workflow := range item.Workflows {
		workflow.Jobs = maps.Clone(workflow.Jobs)
		for jobID, job := range workflow.Jobs {
			parsed, err := reference.ParseNamed(job.SandboxImage)
			if err != nil {
				return item, err
			}
			if _, pinned := parsed.(reference.Digested); pinned {
				continue
			}
			value, ok := cache[job.SandboxImage]
			if !ok {
				value, err = p.digest(job.SandboxImage, false)
				if err != nil {
					return item, err
				}
				cache[job.SandboxImage] = value
			}
			job.SandboxImage = parsed.Name() + "@" + value
			workflow.Jobs[jobID] = job
		}
		item.Workflows[workflowID] = workflow
	}
	return item, nil
}

func requirePinnedImages(item resource.Resource) error {
	for _, workflow := range item.Workflows {
		for _, job := range workflow.Jobs {
			_, digest, found := strings.Cut(job.SandboxImage, "@")
			if !found || !digestPattern.MatchString(digest) {
				return fmt.Errorf("unpinned image: %s", job.SandboxImage)
			}
		}
	}
	return nil
}
