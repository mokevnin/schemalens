package sarif_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/mokevnin/schemalens/internal/finding"
	"github.com/mokevnin/schemalens/internal/sarif"
)

// Manifests (ticket #11) are the only source of positions, and `check` cannot
// take one yet, so the position mapping is tested here at the renderer.
func TestRenderMapsSourcePositionAndValidates(t *testing.T) {
	findings := []finding.Finding{
		{
			Rule:    "column-type-mismatch",
			Object:  finding.Object{Schema: "main", Table: "users", Column: "age"},
			Message: "column age differs",
			Fix:     "change the model",
			Source:  &finding.Position{File: "ent/schema/user.go", Line: 17},
		},
		{
			Rule:    "table-without-primary-key",
			Object:  finding.Object{Schema: "main", Table: "audit_log"},
			Message: "no primary key",
			Fix:     "add one",
		},
	}
	var buf bytes.Buffer
	if err := sarif.Render(&buf, findings, nil); err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if err := compile(t).Validate(doc); err != nil {
		t.Fatalf("does not validate: %v\n%s", err, buf.String())
	}
	out := buf.String()
	for _, want := range []string{
		`"uri": "ent/schema/user.go"`,
		`"startLine": 17`,
		`"ruleId": "column-type-mismatch"`,
		`"fullyQualifiedName": "main.audit_log"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %s\n%s", want, out)
		}
	}
	var again bytes.Buffer
	if err := sarif.Render(&again, findings, nil); err != nil || again.String() != out {
		t.Error("two renders differ")
	}
}

func TestRenderReportsUnmatchedExclusionsAndValidates(t *testing.T) {
	unmatched := []sarif.Unmatched{{Rule: "table-without-primary-key", Object: "main.gone", Reason: "table was dropped"}}
	var buf bytes.Buffer
	if err := sarif.Render(&buf, nil, unmatched); err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if err := compile(t).Validate(doc); err != nil {
		t.Fatalf("does not validate: %v\n%s", err, buf.String())
	}
	for _, want := range []string{`"executionSuccessful": false`, `"toolConfigurationNotifications"`, "main.gone", "table was dropped"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("output lacks %s\n%s", want, buf.String())
		}
	}
	var clean bytes.Buffer
	if err := sarif.Render(&clean, nil, nil); err != nil || strings.Contains(clean.String(), "invocations") {
		t.Errorf("no unmatched exclusions must add no invocations: %v\n%s", err, clean.String())
	}
}

func compile(t *testing.T) *jsonschema.Schema {
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
