// Package gormadapter is the schemalens adapter for gorm. It reads a gorm project's models statically
// (struct tags and field types, through the Go package loader; no project
// code runs) and prints a schemalens manifest on stdout.
//
// Run it in the project directory (the schemalens-adapter-gorm command wraps Run):
//
//	schemalens check --dsn ... --adapter gorm [--adapter-arg ./models]... [--adapter-arg --config=other.yml]
//
// Arguments are Go package patterns (default ./...). --config names the
// override file (default schemalens-gorm.yml in the project, if present).
package gormadapter

import (
	"bytes"
	"fmt"
	"io"

	"github.com/mokevnin/schemalens/adapters/staticmodel"
)

// Run is the adapter's entry point: args are the adapter arguments schemalens
// passes through (--adapter-arg). It prints the manifest to stdout and returns
// the exit code; failures go to stderr and leave stdout empty.
func Run(args []string, stdout, stderr io.Writer) int {
	var joins joinFor
	var doc bytes.Buffer
	code := staticmodel.Run(staticmodel.Adapter[config]{
		Name:       "gorm",
		ConfigFile: defaultConfigFile,
		Packages:   func(c *config) []string { return c.Packages },
		Analyze: func(p *staticmodel.Program, c *config) (*staticmodel.Manifest, error) {
			m, err := analyze(p, c)
			joins = c.joins
			return m, err
		},
	}, args, &doc, stderr)
	if code != 0 {
		return code
	}
	if err := withJoinTableFor(stdout, doc.Bytes(), joins); err != nil {
		_, _ = fmt.Fprintln(stderr, "schemalens-adapter-gorm:", err)
		return 1
	}
	return 0
}
