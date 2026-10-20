package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm/schema"
)

// defaultConfigFile is read from the project directory when --config is not given.
const defaultConfigFile = "schemalens-gorm.yml"

// config supplies what gorm decides at run time and static analysis cannot
// see: the naming strategy and table names set in code.
type config struct {
	// Packages are the Go package patterns to analyse when none are given on
	// the command line. Default: ./...
	Packages []string `yaml:"packages"`
	// Naming mirrors the fields of gorm's schema.NamingStrategy.
	Naming struct {
		TablePrefix   string `yaml:"table_prefix"`
		SingularTable bool   `yaml:"singular_table"`
		NoLowerCase   bool   `yaml:"no_lower_case"`
	} `yaml:"naming"`
	// Tables maps a struct name (or "importpath.Name" when two packages have
	// the same name) to its physical table name, used verbatim. It covers
	// TableName methods that cannot be read statically.
	Tables map[string]string `yaml:"tables"`
	// Exclude lists structs that look like models but are not.
	Exclude []string `yaml:"exclude"`
}

func (c *config) namer() schema.NamingStrategy {
	return schema.NamingStrategy{
		TablePrefix:   c.Naming.TablePrefix,
		SingularTable: c.Naming.SingularTable,
		NoLowerCase:   c.Naming.NoLowerCase,
	}
}

// loadConfig reads path; an empty path means the default file, which is
// optional. An explicitly named file must exist.
func loadConfig(path string) (*config, error) {
	explicit := path != ""
	if !explicit {
		path = defaultConfigFile
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return &config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}
	var c config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return &c, nil
}
