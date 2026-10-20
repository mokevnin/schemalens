package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mokevnin/schemalens/internal/cli"
)

func TestBaselineSilencesCurrentFindings(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")
	applyDDL(t, dbPath, "CREATE TABLE audit_log (message TEXT);")
	cfg := filepath.Join(t.TempDir(), "schemalens.yml")
	dsn := "sqlite://" + dbPath

	if _, stderr, code := run("baseline", "--dsn", dsn, "--config", cfg); code != cli.ExitOK {
		t.Fatalf("baseline exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	got := readFile(t, cfg)
	for _, want := range []string{"table-without-primary-key", "main.audit_log", "reason: baseline"} {
		if !strings.Contains(got, want) {
			t.Errorf("config missing %q:\n%s", want, got)
		}
	}
	if stdout, stderr, code := run("check", "--dsn", dsn, "--config", cfg); code != cli.ExitOK {
		t.Errorf("check after baseline: exit %d, stdout %s stderr %s", code, stdout, stderr)
	}
}

func TestBaselineTwiceDoesNotDuplicate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")
	applyDDL(t, dbPath, "CREATE TABLE audit_log (message TEXT);")
	cfg := filepath.Join(t.TempDir(), "schemalens.yml")
	dsn := "sqlite://" + dbPath
	run("baseline", "--dsn", dsn, "--config", cfg)
	first := readFile(t, cfg)
	if _, stderr, code := run("baseline", "--dsn", dsn, "--config", cfg); code != cli.ExitOK {
		t.Fatalf("second baseline exit code = %d (stderr: %s)", code, stderr)
	}
	if second := readFile(t, cfg); second != first {
		t.Errorf("second baseline changed the file:\n%s\n---\n%s", first, second)
	}
}

func TestBaselineKeepsHandWrittenConfigAndReportsNewViolations(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")
	applyDDL(t, dbPath, "CREATE TABLE audit_log (message TEXT); CREATE TABLE events (message TEXT);")
	cfg := filepath.Join(t.TempDir(), "schemalens.yml")
	dsn := "sqlite://" + dbPath
	writeFile(t, cfg, `# project settings
rules:
  table-without-primary-key:
    enabled: true # keep on

exclusions:
  # raw log, never updated
  - rule: table-without-primary-key
    object: main.audit_log
    reason: append-only raw event log
`)
	if _, stderr, code := run("baseline", "--dsn", dsn, "--config", cfg); code != cli.ExitOK {
		t.Fatalf("baseline exit code = %d (stderr: %s)", code, stderr)
	}
	got := readFile(t, cfg)
	for _, want := range []string{"# project settings", "# keep on", "# raw log, never updated", "reason: append-only raw event log", "main.events", "reason: baseline"} {
		if !strings.Contains(got, want) {
			t.Errorf("config missing %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "main.audit_log") != 1 {
		t.Errorf("audit_log should not be duplicated:\n%s", got)
	}
	if _, _, code := run("check", "--dsn", dsn, "--config", cfg); code != cli.ExitOK {
		t.Errorf("check after baseline: exit %d", code)
	}

	applyDDL(t, dbPath, "CREATE TABLE fresh (message TEXT);")
	stdout, _, code := run("check", "--dsn", dsn, "--config", cfg)
	if code != cli.ExitFindings || !strings.Contains(stdout, "fresh") {
		t.Errorf("new violation: exit %d, stdout %s", code, stdout)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
