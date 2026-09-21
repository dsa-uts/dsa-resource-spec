package publishing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExistingPublishedResources(t *testing.T) {
	// Checked-in releases remain readable after migrating from the Python publisher.
	_, err := readIndex("../..")
	must(t, err)
}

func TestInvalidReleaseIndex(t *testing.T) {
	for _, content := range []string{
		`{}`, `null`, `{"resources":null}`, `{"resources":[]}`,
		`{"resources":{},"unknown":true}`, `{"resources":{"sample":null}}`,
		`{"resources":{}} {}`, `{"resources":{"sample":{"v1.0.0":{}}}}`,
	} {
		t.Run(content, func(t *testing.T) {
			root := t.TempDir()
			must(t, os.Mkdir(filepath.Join(root, "release"), 0755))
			must(t, os.WriteFile(filepath.Join(root, "release/index.json"), []byte(content), 0644))
			if _, err := readIndex(root); err == nil {
				t.Fatal("accepted invalid index")
			}
		})
	}
}

func TestReleaseFilesRequireIndex(t *testing.T) {
	root := t.TempDir()
	index, err := readIndex(root)
	must(t, err)
	if index.Resources == nil {
		t.Fatal("missing empty resource map")
	}
	must(t, writeJSON(filepath.Join(root, "release/sample/v1.0.0.json"), map[string]string{}))
	if _, err := readIndex(root); err == nil {
		t.Fatal("accepted release without index")
	}
}

func TestReleaseSymlinksAreRejected(t *testing.T) {
	for _, target := range []string{"release", "release/index.json", "release/sample", "release/sample/v1.0.0.json"} {
		t.Run(target, func(t *testing.T) {
			f := newFixture(t)
			must(t, f.publisher.publish(f.root))
			f.git(f.root, "pull", "--ff-only")
			path := filepath.Join(f.root, target)
			destination := filepath.Join(t.TempDir(), "original")
			must(t, os.Rename(path, destination))
			must(t, os.Symlink(destination, path))
			if _, err := readIndex(f.root); err == nil {
				t.Fatal("accepted symlink")
			}
		})
	}
}
