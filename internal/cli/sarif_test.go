package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Every golden case, rendered as SARIF, must validate against the vendored
// SARIF 2.1.0 schema.
func TestCheckSarifValidatesAgainstSchema(t *testing.T) {
	schema := compileSarifSchema(t)
	cases, err := filepath.Glob("../../testdata/check/*/schema.sql")
	if err != nil || len(cases) == 0 {
		t.Fatalf("no golden cases found: %v", err)
	}
	for _, schemaFile := range cases {
		dir := filepath.Dir(schemaFile)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			ddl, err := os.ReadFile(schemaFile)
			if err != nil {
				t.Fatal(err)
			}
			dbPath := filepath.Join(t.TempDir(), "db.sqlite")
			applyDDL(t, dbPath, string(ddl))
			stdout, _, _ := run("check", "--dsn", "sqlite://"+dbPath, "--format", "sarif")

			var doc any
			if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&doc); err != nil {
				t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
			}
			if err := schema.Validate(doc); err != nil {
				t.Errorf("SARIF does not validate: %v\n%s", err, stdout)
			}
		})
	}
}

func TestCheckRejectsUnknownFormat(t *testing.T) {
	stdout, stderr, code := run("check", "--dsn", "sqlite://x", "--format", "xml")
	if code != 2 || stdout != "" || !strings.Contains(stderr, "xml") {
		t.Errorf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func compileSarifSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	f, err := os.Open("../../testdata/sarif/sarif-schema-2.1.0.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	doc, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("sarif.json", doc); err != nil {
		t.Fatal(err)
	}
	s, err := c.Compile("sarif.json")
	if err != nil {
		t.Fatal(err)
	}
	return s
}
