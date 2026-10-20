package schemalens_test

import (
	"bytes"
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/mokevnin/schemalens"
)

// sqliteDSN creates a SQLite database from ddl and returns its DSN.
func sqliteDSN(t *testing.T, ddl string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "db.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(ddl); err != nil {
		t.Fatal(err)
	}
	return "sqlite://" + path
}

func TestCheckReportsFindingsForDDL(t *testing.T) {
	dsn := sqliteDSN(t, `CREATE TABLE audit_log (message TEXT)`)
	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Passed() {
		t.Fatal("report with findings must not pass")
	}
	if rep.Dialect != schemalens.SQLite {
		t.Errorf("dialect = %q", rep.Dialect)
	}
	if len(rep.Findings) != 1 || rep.Findings[0].Rule != "table-without-primary-key" {
		t.Fatalf("findings = %+v", rep.Findings)
	}
	want := schemalens.Object{Schema: "main", Table: "audit_log"}
	if rep.Findings[0].Object != want {
		t.Errorf("object = %+v, want %+v", rep.Findings[0].Object, want)
	}
	if rep.Findings[0].Fix == "" {
		t.Error("finding has no fix")
	}
}

func TestCheckCleanSchemaPasses(t *testing.T) {
	dsn := sqliteDSN(t, `CREATE TABLE users (id INTEGER PRIMARY KEY)`)
	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Passed() || len(rep.Findings) != 0 {
		t.Fatalf("report = %+v", rep)
	}
}

func TestExclusionSilencesFindingAndStaleExclusionFails(t *testing.T) {
	dsn := sqliteDSN(t, `CREATE TABLE audit_log (message TEXT)`)
	cfg, err := schemalens.ParseConfig([]byte(`
exclusions:
  - rule: table-without-primary-key
    object: main.audit_log
    reason: append-only raw event log
`))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn, Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Passed() {
		t.Fatalf("excluded finding must pass: %+v", rep)
	}

	stale := sqliteDSN(t, `CREATE TABLE audit_log (id INTEGER PRIMARY KEY)`)
	rep, err = schemalens.Check(context.Background(), schemalens.Options{DSN: stale, Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Passed() || len(rep.Findings) != 0 {
		t.Fatalf("stale exclusion must fail without findings: %+v", rep)
	}
	if len(rep.UnmatchedExclusions) != 1 || rep.UnmatchedExclusions[0].Object != "main.audit_log" ||
		rep.UnmatchedExclusions[0].Reason != "append-only raw event log" {
		t.Fatalf("unmatched = %+v", rep.UnmatchedExclusions)
	}
}

func TestManifestModelRule(t *testing.T) {
	dsn := sqliteDSN(t, `CREATE TABLE users (id BIGINT NOT NULL PRIMARY KEY, email TEXT)`)
	manifest := []byte(`{"version":1,"tables":[{"name":"users","primary_key":["id"],"columns":[
		{"name":"id","required":true},
		{"name":"email","required":true,"source":{"file":"user.go","line":18}}]}]}`)
	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn, Manifest: manifest})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Findings) != 1 || rep.Findings[0].Rule != "required-field-nullable-column" {
		t.Fatalf("findings = %+v", rep.Findings)
	}
	src := rep.Findings[0].Source
	if src == nil || src.File != "user.go" || src.Line != 18 {
		t.Errorf("source = %+v", src)
	}
}

func TestParsedModelEqualsRawManifest(t *testing.T) {
	dsn := sqliteDSN(t, `CREATE TABLE users (id BIGINT NOT NULL PRIMARY KEY, email TEXT)`)
	raw := []byte(`{"version":1,"tables":[{"name":"users","columns":[{"name":"email","required":true}]}]}`)
	model, err := schemalens.ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn, Model: model})
	if err != nil || len(rep.Findings) != 1 {
		t.Fatalf("rep = %+v, err = %v", rep, err)
	}
	_, err = schemalens.Check(context.Background(), schemalens.Options{DSN: dsn, Model: model, Manifest: raw})
	if err == nil {
		t.Fatal("Model and Manifest together must be an error")
	}
}

func TestInvalidManifestIsAnError(t *testing.T) {
	dsn := sqliteDSN(t, `CREATE TABLE users (id INTEGER PRIMARY KEY)`)
	_, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn, Manifest: []byte(`{"version":99}`)})
	if err == nil || !strings.Contains(err.Error(), "version 99 is not supported") {
		t.Fatalf("err = %v", err)
	}
	if _, err := schemalens.ParseManifest([]byte(`not json`)); err == nil {
		t.Fatal("ParseManifest accepted garbage")
	}
}

func TestBadDSNIsAnError(t *testing.T) {
	if _, err := schemalens.Check(context.Background(), schemalens.Options{DSN: "nope"}); err == nil {
		t.Fatal("want error")
	}
}

func TestSkippedRulesAndWarnings(t *testing.T) {
	dsn := sqliteDSN(t, `CREATE TABLE users (id INTEGER PRIMARY KEY)`)
	manifest := []byte(`{"version":1,"tables":[{"name":"ghost","columns":[{"name":"id"}]}]}`)
	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn, Manifest: manifest})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Warnings) == 0 {
		t.Error("manifest table missing from the database must warn")
	}
	if len(rep.SkippedRules) == 0 {
		t.Error("PostgreSQL-only rules must be reported as skipped on SQLite")
	}
}

func TestConfigRejectsUnknownRule(t *testing.T) {
	if _, err := schemalens.ParseConfig([]byte("rules:\n  nope:\n    enabled: false\n")); err == nil {
		t.Fatal("want error")
	}
	if _, err := schemalens.LoadConfig(filepath.Join(t.TempDir(), "missing.yml")); err == nil {
		t.Fatal("explicit missing file must fail")
	}
}

func TestProgrammaticConfigDisablesRule(t *testing.T) {
	dsn := sqliteDSN(t, `CREATE TABLE audit_log (message TEXT)`)
	off := false
	cfg := &schemalens.Config{Rules: map[string]schemalens.RuleConfig{
		"table-without-primary-key": {Enabled: &off},
	}}
	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn, Config: cfg})
	if err != nil || !rep.Passed() {
		t.Fatalf("rep = %+v, err = %v", rep, err)
	}
	bad := &schemalens.Config{Rules: map[string]schemalens.RuleConfig{"nope": {}}}
	if _, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn, Config: bad}); err == nil {
		t.Fatal("invalid programmatic config must fail")
	}
}

func TestWriteJSONAndSARIF(t *testing.T) {
	dsn := sqliteDSN(t, `CREATE TABLE audit_log (message TEXT)`)
	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	var js, sarif bytes.Buffer
	if err := rep.WriteJSON(&js); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(js.String(), `"rule": "table-without-primary-key"`) || strings.Contains(js.String(), "warnings") {
		t.Errorf("json = %s", js.String())
	}
	if err := rep.WriteSARIF(&sarif); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sarif.String(), `"version": "2.1.0"`) || !strings.Contains(sarif.String(), "table-without-primary-key") {
		t.Errorf("sarif = %s", sarif.String())
	}
}

func TestRulesAndExplain(t *testing.T) {
	rs := schemalens.Rules()
	if len(rs) == 0 {
		t.Fatal("no rules")
	}
	var found bool
	for i, r := range rs {
		if i > 0 && rs[i-1].ID >= r.ID {
			t.Errorf("rules not sorted at %s", r.ID)
		}
		if r.ID == "plain-tables" && !r.DefaultOn {
			found = true
		}
	}
	if !found {
		t.Error("plain-tables must be listed as off by default")
	}
	e, ok := schemalens.Explain("table-without-primary-key")
	if !ok || e.Why == "" || e.Fix == "" || e.Exception == "" || e.ID != "table-without-primary-key" || len(e.Dialects) != 3 {
		t.Fatalf("explanation = %+v ok=%v", e, ok)
	}
	if _, ok := schemalens.Explain("nope"); ok {
		t.Fatal("unknown rule explained")
	}
}

func TestBaselineWritesExclusionsAndIsIdempotent(t *testing.T) {
	dsn := sqliteDSN(t, `CREATE TABLE audit_log (message TEXT)`)
	out, n, err := schemalens.Baseline(context.Background(), schemalens.BaselineOptions{DSN: dsn})
	if err != nil || n != 1 {
		t.Fatalf("n = %d, err = %v", n, err)
	}
	if !strings.Contains(string(out), "object: main.audit_log") || !strings.Contains(string(out), "reason: baseline") {
		t.Fatalf("yaml = %s", out)
	}
	again, n, err := schemalens.Baseline(context.Background(), schemalens.BaselineOptions{DSN: dsn, Config: out})
	if err != nil || n != 0 || !bytes.Equal(again, out) {
		t.Fatalf("second run n = %d err = %v", n, err)
	}
}
