package resource

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/distribution/reference"
	"gopkg.in/yaml.v3"
)

// Archive writes the complete resource tree as a ZIP. Resolved maps each tagged
// image reference to a digest reference in the same repository. Digest references
// already present in the definition are retained. No registry access occurs.
// On error the caller must discard any partial output.
func Archive(root fs.FS, out io.Writer, resolved map[string]string) error {
	resource, err := Read(root)
	if err != nil {
		return err
	}
	for workflowID, workflow := range resource.Definition.Workflows {
		for jobID, job := range workflow.Jobs {
			original, _ := imageReference(job.SandboxImage)
			if _, ok := original.(reference.Digested); ok {
				continue
			}
			pinned, ok := resolved[job.SandboxImage]
			if !ok {
				return fmt.Errorf("unresolved image: %s", job.SandboxImage)
			}
			ref, err := imageReference(pinned)
			if err != nil {
				return err
			}
			digest, ok := ref.(reference.Digested)
			if !ok || digest.Digest().Algorithm().String() != "sha256" {
				return fmt.Errorf("resolved image must use sha256 digest: %s", pinned)
			}
			if ref.Name() != original.Name() {
				return fmt.Errorf("resolved image repository differs: %s", pinned)
			}
			if _, tagged := ref.(reference.Tagged); tagged {
				return fmt.Errorf("resolved image must omit tag: %s", pinned)
			}
			job.SandboxImage = pinned
			workflow.Jobs[jobID] = job
		}
		resource.Definition.Workflows[workflowID] = workflow
	}
	definition, err := yaml.Marshal(resource.Definition)
	if err != nil {
		return err
	}
	archive := zip.NewWriter(out)
	err = fs.WalkDir(root, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		if err := relative(name); err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s: symlink is forbidden", name)
		}
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		if entry.IsDir() {
			header.Name += "/"
			header.SetMode(0755 | fs.ModeDir)
			_, err := archive.CreateHeader(header)
			return err
		}
		data, err := readRegular(root, name)
		if err != nil {
			return err
		}
		if name == "resource.yaml" {
			data = definition
		}
		mode := fs.FileMode(0644)
		if info.Mode()&0111 != 0 {
			mode = 0755
		}
		header.SetMode(mode)
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		return err
	})
	if err != nil {
		_ = archive.Close()
		return err
	}
	return archive.Close()
}
