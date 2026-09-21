// resource-ci runs this repository's GitHub validation and publication tasks.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dsa-uts/dsa-resource-spec/internal/publishing"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: resource-ci <check|build-images|publish> [options]")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	root := flags.String("root", ".", "Manifest directory")
	var base *string
	var ghaCache *bool
	switch args[0] {
	case "check":
		base = flags.String("base", "", "Base commit to compare")
	case "build-images":
		ghaCache = flags.Bool("gha-cache", false, "Use GitHub Actions image cache")
	case "publish":
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	directory, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	switch args[0] {
	case "check":
		if *base == "" {
			return fmt.Errorf("check requires --base")
		}
		return publishing.Check(directory, *base)
	case "build-images":
		return publishing.BuildImages(directory, *ghaCache)
	default:
		return publishing.Publish(directory)
	}
}
