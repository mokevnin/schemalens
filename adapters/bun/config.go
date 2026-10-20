package bunadapter

// defaultConfigFile is read from the project directory when --config is not given.
const defaultConfigFile = "schemalens-bun.yml"

// config supplies what bun decides at run time and static analysis cannot
// see: the table-name inflector and table names set in code.
type config struct {
	// Packages are the Go package patterns to analyse when none are given on
	// the command line. Default: ./...
	Packages []string `yaml:"packages"`
	// Naming mirrors what the project does to bun's naming at start-up.
	Naming struct {
		// SingularTable is set when the project calls
		// schema.SetTableNameInflector with an identity function: tables are
		// named after the struct, not pluralized.
		SingularTable bool `yaml:"singular_table"`
	} `yaml:"naming"`
	// Tables maps a struct name (or "importpath.Name" when two packages have
	// the same name) to its physical table name, used verbatim. It covers
	// table names that are set in code (a ModelTableExpr, say) rather than in
	// the struct tag.
	Tables map[string]string `yaml:"tables"`
	// Exclude lists structs that look like models but are not.
	Exclude []string `yaml:"exclude"`
}
