package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	resource "github.com/dsa-uts/dsa-resource-spec"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: resource-spec <validate|catalog> MANIFEST_DIRECTORY | show MANIFEST_DIRECTORY RESOURCE_ID")
	}
	if args[0] != "validate" && args[0] != "catalog" && args[0] != "show" {
		return fmt.Errorf("unknown command %q", args[0])
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	count := 1
	if args[0] == "show" {
		count = 2
	}
	if flags.NArg() != count {
		return fmt.Errorf("%s expects %d arguments", args[0], count)
	}
	manifest, err := resource.LoadManifest(flags.Arg(0))
	if err != nil {
		return err
	}
	switch args[0] {
	case "validate":
		return nil
	case "catalog":
		catalog := struct {
			Resources     []resource.Metadata            `json:"resources"`
			SandboxImages map[string]resource.ImageBuild `json:"sandbox-images"`
		}{
			Resources:     make([]resource.Metadata, 0, len(manifest.Resources)),
			SandboxImages: manifest.SandboxImages,
		}
		for _, item := range manifest.Resources {
			catalog.Resources = append(catalog.Resources, item.Metadata)
		}
		return json.NewEncoder(out).Encode(catalog)
	default:
		for _, item := range manifest.Resources {
			if item.Metadata.ID == flags.Arg(1) {
				return json.NewEncoder(out).Encode(item)
			}
		}
		return fmt.Errorf("unknown resource ID %q", flags.Arg(1))
	}
}
