package cli_test

import (
	"encoding/json"
	"testing"

	"github.com/mokevnin/schemalens/internal/cli"
)

func TestManifestSchemaCommandPrintsJSONSchema(t *testing.T) {
	stdout, stderr, code := run("manifest-schema")
	if code != cli.ExitOK {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	var doc struct {
		Schema     string                    `json:"$schema"`
		Properties map[string]map[string]any `json:"properties"`
		Defs       map[string]struct {
			Properties map[string]any `json:"properties"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if doc.Schema == "" {
		t.Error("printed document declares no $schema")
	}
	for _, p := range []string{"version", "tables"} {
		if _, ok := doc.Properties[p]; !ok {
			t.Errorf("schema has no %q property", p)
		}
	}
	for def, props := range map[string][]string{
		"column": {"required", "unique", "max_length", "enum"},
		"table":  {"uniques", "relations", "join_table_for"},
	} {
		for _, p := range props {
			if _, ok := doc.Defs[def].Properties[p]; !ok {
				t.Errorf("schema $defs.%s has no %q property", def, p)
			}
		}
	}
}
