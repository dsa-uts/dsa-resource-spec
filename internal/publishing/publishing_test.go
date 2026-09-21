package publishing

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	resource "github.com/dsa-uts/dsa-resource-spec/v2"
)

const testImage = "ghcr.io/example/default"

var digestOne = "sha256:" + strings.Repeat("1", 64)
var digestTwo = "sha256:" + strings.Repeat("2", 64)

// Git and resource loading are real. Only Docker/registry operations are replaced.
// The fake stores registry state and records commands independently of publication.
type fakeRegistry struct {
	images  map[string]string
	built   string
	failure string
	calls   [][]string
}

func (r *fakeRegistry) run(directory string, stream bool, name string, args ...string) (commandResult, error) {
	if name != "docker" && name != "regctl" {
		return execute(directory, stream, name, args...)
	}
	r.calls = append(r.calls, append([]string{name}, args...))
	if name == "docker" {
		if r.failure == "build" {
			return commandResult{}, errors.New("build failed")
		}
		output := args[slices.Index(args, "--output")+1]
		var layout string
		for _, part := range strings.Split(output, ",") {
			if value, ok := strings.CutPrefix(part, "dest="); ok {
				layout = value
			}
		}
		manifests := []map[string]string{{"digest": r.built}}
		if r.failure == "invalid-export" {
			manifests = nil
		}
		return commandResult{}, writeJSON(filepath.Join(layout, "index.json"), map[string]any{"manifests": manifests})
	}
	switch args[1] {
	case "digest":
		image := args[2]
		local := strings.HasPrefix(image, "ocidir://")
		if !local && (r.failure == "auth" || r.failure == "network") {
			message := "unauthorized: authentication required"
			if r.failure == "network" {
				message = "connection refused"
			}
			return commandResult{stderr: message}, errors.New(message)
		}
		value := r.images[image]
		if local {
			value = r.built
		}
		if value == "" {
			return commandResult{stderr: "manifest unknown: not found"}, errors.New("not found")
		}
		return commandResult{stdout: value + "\n"}, nil
	case "copy":
		if r.failure == "copy" {
			return commandResult{}, errors.New("upload failed")
		}
		source, destination := args[2], args[3]
		value := r.built
		if !strings.HasPrefix(source, "ocidir://") {
			_, value, _ = strings.Cut(source, "@")
		}
		if r.failure == "digest-mismatch" {
			value = digestOne
		}
		r.images[destination] = value
		return commandResult{}, nil
	default:
		return commandResult{}, fmt.Errorf("unexpected command: %s %v", name, args)
	}
}

type fixture struct {
	t                     *testing.T
	root, remote, initial string
	registry              *fakeRegistry
	publisher             publisher
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// Make Git's retry classification independent of the developer's locale.
	t.Setenv("LC_ALL", "C")
	temp := t.TempDir()
	f := &fixture{t: t, root: filepath.Join(temp, "source"), remote: filepath.Join(temp, "remote.git")}
	must(t, os.CopyFS(f.root, os.DirFS("../../testdata/cli/valid/basic/input")))
	f.registry = &fakeRegistry{images: map[string]string{testImage + ":latest": digestOne}, built: digestOne}
	f.publisher = publisher{f.registry.run}
	f.git(f.root, "init", "-b", "main")
	f.git(f.root, "config", "user.name", "Test")
	f.git(f.root, "config", "user.email", "test@example.com")
	f.initial = f.commit()
	f.git(temp, "init", "--bare", "--initial-branch=main", f.remote)
	f.git(f.root, "remote", "add", "origin", f.remote)
	f.git(f.root, "push", "-u", "origin", "main")
	return f
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func wantError(t *testing.T, err error, message string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), message) {
		t.Fatalf("want error containing %q, got %v", message, err)
	}
}
func (f *fixture) git(root string, args ...string) string {
	f.t.Helper()
	output, err := f.publisher.git(root, args...)
	must(f.t, err)
	return output
}
func (f *fixture) commit() string {
	f.git(f.root, "add", ".")
	f.git(f.root, "commit", "-m", "Test change")
	return f.git(f.root, "rev-parse", "HEAD")
}
func (f *fixture) write(relative, content string) {
	f.t.Helper()
	path := filepath.Join(f.root, relative)
	must(f.t, os.MkdirAll(filepath.Dir(path), 0755))
	must(f.t, os.WriteFile(path, []byte(content), 0644))
}
func (f *fixture) replace(relative, before, after string) {
	f.t.Helper()
	data, err := os.ReadFile(filepath.Join(f.root, relative))
	must(f.t, err)
	f.write(relative, strings.ReplaceAll(string(data), before, after))
}
func (f *fixture) bump(version string)           { f.replace("sample/resource.yaml", "v1.0.0", version) }
func (f *fixture) remoteFile(path string) string { return f.git(f.remote, "show", "main:"+path) }
func (f *fixture) published(version, digest string) {
	f.t.Helper()
	data := f.remoteFile("release/sample/" + version + ".json")
	item, err := resource.DecodeResource(strings.NewReader(data))
	must(f.t, err)
	for _, workflow := range item.Workflows {
		for _, job := range workflow.Jobs {
			if job.SandboxImage != testImage+"@"+digest {
				f.t.Fatalf("unexpected image: %s", job.SandboxImage)
			}
		}
	}
}
func (f *fixture) setupBuild() {
	f.write("sandbox/Dockerfile", "FROM scratch\n")
	f.write("manifest.yaml", `resources:
  - id: sample
    path: sample
sandbox-images:
  default:
    context: sandbox
    dockerfile: sandbox/Dockerfile
    image: ghcr.io/example/default
    platforms: [linux/amd64, linux/arm64]
`)
}

func TestPublishAndRerun(t *testing.T) {
	f := newFixture(t)
	must(t, f.publisher.publish(f.root))
	f.published("v1.0.0", digestOne)
	var index releaseIndex
	must(t, json.Unmarshal([]byte(f.remoteFile("release/index.json")), &index))
	if index.Resources["sample"]["v1.0.0"].SourceCommit != f.initial {
		t.Fatal("wrong source commit")
	}
	if len(f.registry.calls) != 1 {
		t.Fatalf("tag resolved more than once: %v", f.registry.calls)
	}
	head := f.git(f.remote, "rev-parse", "main")
	f.registry.failure = "auth"
	f.registry.calls = nil
	must(t, f.publisher.publish(f.root))
	if f.git(f.remote, "rev-parse", "main") != head || len(f.registry.calls) != 0 {
		t.Fatal("rerun changed publication or accessed registry")
	}
}

func TestResolutionFailurePublishesNothing(t *testing.T) {
	f := newFixture(t)
	f.registry.images = map[string]string{}
	wantError(t, f.publisher.publish(f.root), "cannot resolve")
	if f.git(f.remote, "rev-parse", "main") != f.initial {
		t.Fatal("published after resolution failure")
	}
}

func TestPinnedImageNeedsNoResolution(t *testing.T) {
	f := newFixture(t)
	f.replace("sample/resource.yaml", testImage+":latest", testImage+"@"+digestOne)
	f.commit()
	f.git(f.root, "push", "origin", "main")
	f.registry.failure = "auth"
	must(t, f.publisher.publish(f.root))
	if len(f.registry.calls) != 0 {
		t.Fatal("resolved pinned image")
	}
}

func TestPublishSkipsChangedSourceUntilVersionChanges(t *testing.T) {
	f := newFixture(t)
	must(t, f.publisher.publish(f.root))
	f.git(f.root, "pull", "--ff-only")
	original := f.remoteFile("release/sample/v1.0.0.json")
	originalIndex := f.remoteFile("release/index.json")
	f.write("sample/expected.txt", "changed")
	f.replace("sample/resource.yaml", "resource:", "# edited definition\nresource:")
	f.commit()
	f.git(f.root, "push", "origin", "main")
	must(t, Check(f.root))
	head := f.git(f.remote, "rev-parse", "main")
	f.registry.failure = "auth"
	f.registry.calls = nil
	must(t, f.publisher.publish(f.root))
	if f.git(f.remote, "rev-parse", "main") != head || len(f.registry.calls) != 0 {
		t.Fatal("same-version changes caused publication or registry access")
	}
	if f.remoteFile("release/index.json") != originalIndex {
		t.Fatal("changed existing index entry")
	}
	f.bump("v1.0.1")
	sourceCommit := f.commit()
	f.git(f.root, "push", "origin", "main")
	f.registry.failure = ""
	must(t, Check(f.root))
	must(t, f.publisher.publish(f.root))
	if f.remoteFile("release/sample/v1.0.0.json") != original {
		t.Fatal("overwrote previous release")
	}
	manifest, err := resource.LoadManifest(f.root)
	must(t, err)
	pinned, err := f.publisher.pinImages(manifest.Resources[0], map[string]string{})
	must(t, err)
	var published resource.Resource
	must(t, json.Unmarshal([]byte(f.remoteFile("release/sample/v1.0.1.json")), &published))
	if !reflect.DeepEqual(published, pinned) {
		t.Fatal("new release does not match updated source")
	}
	var index releaseIndex
	must(t, json.Unmarshal([]byte(f.remoteFile("release/index.json")), &index))
	if len(index.Resources["sample"]) != 2 || index.Resources["sample"]["v1.0.1"].SourceCommit != sourceCommit {
		t.Fatal("new version was not appended with its source commit")
	}
}

func TestQueuedSnapshotsBothPublish(t *testing.T) {
	f := newFixture(t)
	f.bump("v1.1.0")
	second := f.commit()
	f.git(f.root, "push", "origin", "main")
	f.git(f.root, "checkout", "--detach", f.initial)
	must(t, f.publisher.publish(f.root))
	f.git(f.root, "checkout", "--detach", second)
	f.registry.images[testImage+":latest"] = digestTwo
	must(t, f.publisher.publish(f.root))
	f.published("v1.0.0", digestOne)
	f.published("v1.1.0", digestTwo)
	if !strings.Contains(f.remoteFile("sample/resource.yaml"), "v1.1.0") {
		t.Fatal("overwrote main source")
	}
}

func TestPushRacePreservesMainAndOriginalDigest(t *testing.T) {
	f := newFixture(t)
	original := f.publisher.run
	raced := false
	f.publisher.run = func(directory string, stream bool, name string, args ...string) (commandResult, error) {
		if name == "git" && len(args) >= 3 && reflect.DeepEqual(args[:3], []string{"push", "origin", "HEAD:refs/heads/main"}) && !raced {
			raced = true
			f.bump("v1.1.0")
			f.commit()
			f.git(f.root, "push", "origin", "main")
			f.registry.images[testImage+":latest"] = digestTwo
		}
		return original(directory, stream, name, args...)
	}
	must(t, f.publisher.publish(f.root))
	if !raced || !strings.Contains(f.remoteFile("sample/resource.yaml"), "v1.1.0") {
		t.Fatal("race lost main source")
	}
	f.published("v1.0.0", digestOne)
	if len(f.registry.calls) != 1 {
		t.Fatal("retry resolved tag again")
	}
}

func TestBuildImages(t *testing.T) {
	for _, scenario := range []string{"unchanged", "changed", "first"} {
		t.Run(scenario, func(t *testing.T) {
			f := newFixture(t)
			f.setupBuild()
			if scenario == "changed" {
				f.registry.built = digestTwo
			}
			if scenario == "first" {
				f.registry.images = map[string]string{}
			}
			must(t, f.publisher.buildImages(f.root, true))
			var copies [][]string
			for _, call := range f.registry.calls {
				if call[0] == "regctl" && call[2] == "copy" {
					copies = append(copies, call)
				}
			}
			if scenario == "unchanged" {
				if len(copies) != 0 {
					t.Fatal("pushed unchanged image")
				}
				return
			}
			if len(copies) != 2 {
				t.Fatalf("expected retention and latest copies: %v", copies)
			}
			fixed := copies[0][4]
			pattern := `:\d{14}-` + strings.ReplaceAll(f.registry.built, ":", "-") + `$`
			if !regexp.MustCompile(pattern).MatchString(fixed) || copies[1][4] != testImage+":latest" {
				t.Fatalf("wrong publication order: %v", copies)
			}
			if f.registry.images[testImage+":latest"] != f.registry.built {
				t.Fatal("latest digest mismatch")
			}
		})
	}
}

func TestBuildFailuresStopPublication(t *testing.T) {
	for _, failure := range []string{"build", "auth", "network", "copy", "digest-mismatch", "invalid-export"} {
		t.Run(failure, func(t *testing.T) {
			f := newFixture(t)
			f.setupBuild()
			f.registry.built = digestTwo
			f.registry.failure = failure
			if err := f.publisher.buildImages(f.root, true); err == nil {
				t.Fatal("build should fail")
			}
			if f.registry.images[testImage+":latest"] != digestOne {
				t.Fatal("moved latest after failure")
			}
		})
	}
}

func TestImagePinningPreservesPortAndSource(t *testing.T) {
	f := newFixture(t)
	manifest, err := resource.LoadManifest(f.root)
	must(t, err)
	item := manifest.Resources[0]
	image := "registry.example:5000/sandbox:release"
	for _, workflow := range item.Workflows {
		for id, job := range workflow.Jobs {
			job.SandboxImage = image
			workflow.Jobs[id] = job
		}
	}
	f.registry.images[image] = digestOne
	pinned, err := f.publisher.pinImages(item, map[string]string{})
	must(t, err)
	for workflowID, workflow := range pinned.Workflows {
		for jobID, job := range workflow.Jobs {
			if job.SandboxImage != "registry.example:5000/sandbox@"+digestOne {
				t.Fatal("lost registry port")
			}
			if item.Workflows[workflowID].Jobs[jobID].SandboxImage != image {
				t.Fatal("mutated source resource")
			}
		}
	}
}
