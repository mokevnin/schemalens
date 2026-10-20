// Command schemalens-adapter-gorm reads a gorm project's models statically
// (struct tags and field types, through the Go package loader; no project
// code runs) and prints a schemalens manifest on stdout.
//
// Run it in the project directory:
//
//	schemalens check --dsn ... --adapter gorm [--adapter-arg ./models]... [--adapter-arg --config=other.yml]
//
// Arguments are Go package patterns (default ./...). --config names the
// override file (default schemalens-gorm.yml in the project, if present).
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/mokevnin/schemalens/adapters/staticmodel"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
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
