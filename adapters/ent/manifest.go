package entadapter

// The adapter keeps its own copy of the manifest wire format on purpose: it
// is a separate module and must not import the core. The contract is the JSON
// Schema printed by `schemalens manifest-schema`, and the tests validate the
// adapter's output with the real `schemalens check --manifest`.

// ManifestVersion is the manifest format version this adapter prints.
const ManifestVersion = 1

type manifest struct {
	Version int     `json:"version"`
	Tables  []table `json:"tables"`
}

type table struct {
	Schema       string     `json:"schema,omitempty"`
	Name         string     `json:"name"`
	Source       *position  `json:"source,omitempty"`
	Columns      []column   `json:"columns"`
	PrimaryKey   []string   `json:"primary_key,omitempty"`
	JoinTableFor []tableRef `json:"join_table_for,omitempty"`
	Uniques      []unique   `json:"uniques,omitempty"`
	Relations    []relation `json:"relations,omitempty"`
}

type unique struct {
	Columns []string `json:"columns"`
}

type relation struct {
	Columns    []string `json:"columns"`
	References tableRef `json:"references"`
	RefColumns []string `json:"ref_columns,omitempty"`
}

type tableRef struct {
	Schema string `json:"schema,omitempty"`
	Table  string `json:"table"`
}

type column struct {
	Name     string    `json:"name"`
	Required bool      `json:"required,omitempty"`
	Unique   bool      `json:"unique,omitempty"`
	MaxLen   int       `json:"max_length,omitempty"`
	Enum     []string  `json:"enum,omitempty"`
	Source   *position `json:"source,omitempty"`
}

type position struct {
	File string `json:"file"`
	Line int    `json:"line"`
}
