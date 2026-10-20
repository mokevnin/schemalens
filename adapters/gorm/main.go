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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/mokevnin/schemalens/adapters/staticmodel"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("schemalens-adapter-gorm", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "override file (default "+defaultConfigFile+" if present)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "schemalens-adapter-gorm:", err)
		return 2
	}
	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = cfg.Packages
	}
	prog, err := staticmodel.Load(".", patterns...)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "schemalens-adapter-gorm:", err)
		return 1
	}
	m, err := analyze(prog, cfg)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "schemalens-adapter-gorm:", err)
		return 1
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "schemalens-adapter-gorm:", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s\n", out)
	return 0
}
