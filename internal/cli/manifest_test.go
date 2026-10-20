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
}
