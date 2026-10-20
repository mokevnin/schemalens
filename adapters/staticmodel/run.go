package staticmodel

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig reads the YAML override file into a C. An empty path means
// defaultPath, which is optional; an explicitly named file must exist.
// Unknown keys are an error.
func LoadConfig[C any](path, defaultPath string) (*C, error) {
	explicit := path != ""
	if !explicit {
		path = defaultPath
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return new(C), nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}
	c := new(C)
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(c); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return c, nil
}

// Adapter describes a static adapter to Run.
type Adapter[C any] struct {
	// Name is the adapter name: "gorm" for the binary schemalens-adapter-gorm.
	Name string
	// ConfigFile is the default override file, read from the project
	// directory when --config is not given.
	ConfigFile string
	// Packages returns the package patterns configured in the override file.
	Packages func(*C) []string
	// Analyze turns the loaded project into a manifest.
	Analyze func(*Program, *C) (*Manifest, error)
}

// Run is the whole adapter command: parse --config and package patterns from
// args, load the project in the working directory, analyse it and print the
// manifest on stdout. It returns the process exit code: 0, 1 for an analysis
// failure, 2 for bad usage or configuration.
func Run[C any](a Adapter[C], args []string, stdout, stderr io.Writer) int {
	prefix := "schemalens-adapter-" + a.Name
	fs := flag.NewFlagSet(prefix, flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "override file (default "+a.ConfigFile+" if present)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := LoadConfig[C](*configPath, a.ConfigFile)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, prefix+":", err)
		return 2
	}
	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = a.Packages(cfg)
	}
	prog, err := Load(".", patterns...)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, prefix+":", err)
		return 1
	}
	m, err := a.Analyze(prog, cfg)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, prefix+":", err)
		return 1
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintln(stderr, prefix+":", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s\n", out)
	return 0
}
