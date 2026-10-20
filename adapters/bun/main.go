// Command schemalens-adapter-bun reads a bun project's models statically
// (struct tags and field types, through the Go package loader; no project
// code runs) and prints a schemalens manifest on stdout.
//
// Run it in the project directory:
//
//	schemalens check --dsn ... --adapter bun [--adapter-arg ./models]... [--adapter-arg --config=other.yml]
//
// Arguments are Go package patterns (default ./...). --config names the
// override file (default schemalens-bun.yml in the project, if present).
package main

import (
	"os"

	"github.com/mokevnin/schemalens/adapters/staticmodel"
)

func main() {
	os.Exit(staticmodel.Run(staticmodel.Adapter[config]{
		Name:       "bun",
		ConfigFile: defaultConfigFile,
		Packages:   func(c *config) []string { return c.Packages },
		Analyze:    analyze,
	}, os.Args[1:], os.Stdout, os.Stderr))
}
