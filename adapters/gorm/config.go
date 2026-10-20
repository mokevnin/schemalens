package main

import (
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
	// TableName methods that cannot be read statically. A many2many name
	// ("many2many:enrollments") may be a key too, for a join table renamed
	// at run time.
	Tables map[string]string `yaml:"tables"`
	// Exclude lists structs that look like models but are not.
	Exclude []string `yaml:"exclude"`

	// joins is the analysis' result for main: which tables are join tables.
	joins joinFor
}

func (c *config) namer() schema.NamingStrategy {
	return schema.NamingStrategy{
		TablePrefix:   c.Naming.TablePrefix,
		SingularTable: c.Naming.SingularTable,
		NoLowerCase:   c.Naming.NoLowerCase,
	}
}
