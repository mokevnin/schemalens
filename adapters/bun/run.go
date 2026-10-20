// Package bunadapter is the schemalens adapter for bun. It reads a bun project's models statically
// (struct tags and field types, through the Go package loader; no project
// code runs) and prints a schemalens manifest on stdout.
//
// Run it in the project directory (the schemalens-adapter-bun command wraps Run):
//
//	schemalens check --dsn ... --adapter bun [--adapter-arg ./models]... [--adapter-arg --config=other.yml]
//
// Arguments are Go package patterns (default ./...). --config names the
// override file (default schemalens-bun.yml in the project, if present).
package bunadapter

import (
	"io"

	"github.com/mokevnin/schemalens/adapters/staticmodel"
)

// Run is the adapter's entry point: args are the adapter arguments schemalens
// passes through (--adapter-arg). It prints the manifest to stdout and returns
// the exit code; failures go to stderr and leave stdout empty.
func Run(args []string, stdout, stderr io.Writer) int {
	return staticmodel.Run(staticmodel.Adapter[config]{
		Name:       "bun",
		ConfigFile: defaultConfigFile,
		Packages:   func(c *config) []string { return c.Packages },
		Analyze:    analyze,
	}, args, stdout, stderr)
}
