package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/mod/semver"

	resource "github.com/dsa-uts/dsa-resource-spec"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 3 && args[0] == "compare" {
		if !semver.IsValid(args[1]) || !semver.IsValid(args[2]) {
			return fmt.Errorf("invalid version")
		}
		_, err := fmt.Fprintln(out, semver.Compare(args[1], args[2]))
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("usage: resource-spec <validate|inspect|manifest|archive|compare> [flags] RESOURCE_DIRECTORY")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	output := flags.String("output", "", "ZIP destination (outside the resource directory)")
	images := flags.String("images", "", "JSON map of original image references to resolved digest references")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("expected one resource directory")
	}
	root := os.DirFS(flags.Arg(0))
	switch args[0] {
	case "manifest":
		m, err := resource.ReadManifest(root)
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(m)
	case "validate":
		return resource.Validate(root)
	case "inspect":
		r, err := resource.Read(root)
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(r.Definition)
	case "archive":
		return writeArchive(flags.Arg(0), *output, *images)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func writeArchive(sourceDirectory, output, images string) error {
	root := os.DirFS(sourceDirectory)
	if output == "" {
		return fmt.Errorf("--output is required")
	}
	source, err := filepath.EvalSymlinks(sourceDirectory)
	if err != nil {
		return err
	}
	source, err = filepath.Abs(source)
	if err != nil {
		return err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(output))
	if err != nil {
		return err
	}
	parent, err = filepath.Abs(parent)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(source, parent)
	if err != nil {
		return err
	}
	if rel == "." || filepath.IsLocal(rel) {
		return fmt.Errorf("output must be outside resource directory")
	}
	resolved := map[string]string{}
	if images != "" {
		b, err := os.ReadFile(images)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(b, &resolved); err != nil {
			return err
		}
	}
	file, err := os.CreateTemp(parent, ".resource-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	err = resource.Archive(root, file, resolved)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), output)
}
