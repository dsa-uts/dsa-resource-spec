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
		return fmt.Errorf("usage: resource-spec <validate|inspect|manifest> RESOURCE_DIRECTORY")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
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
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
