package cli_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mokevnin/schemalens/internal/cli"
)

func TestCheckRejectsBadConfig(t *testing.T) {
	tests := map[string]string{
		"exclusion without reason": "exclusions:\n  - rule: table-without-primary-key\n    object: main.audit_log\n",
		"exclusion blank reason":   "exclusions:\n  - rule: table-without-primary-key\n    object: main.audit_log\n    reason: \"  \"\n",
		"exclusion without rule":   "exclusions:\n  - object: main.audit_log\n    reason: x\n",
		"exclusion unknown rule":   "exclusions:\n  - rule: no-such-rule\n    object: main.t\n    reason: x\n",
		"exclusion bad object":     "exclusions:\n  - rule: table-without-primary-key\n    object: audit_log\n    reason: x\n",
		"exclusion bad kind":       "exclusions:\n  - rule: table-without-primary-key\n    object: main.t.view:v\n    reason: x\n",
		"unknown rule section":     "rules:\n  no-such-rule:\n    enabled: false\n",
		"unknown top-level key":    "exclusion: []\n",
		"not yaml":                 "rules: [\n",
	}
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")
	applyDDL(t, dbPath, "CREATE TABLE audit_log (message TEXT);")
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := filepath.Join(t.TempDir(), "schemalens.yml")
			writeFile(t, cfg, content)
			stdout, stderr, code := run("check", "--dsn", "sqlite://"+dbPath, "--config", cfg)
			if code != cli.ExitUsage {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, cli.ExitUsage, stderr)
			}
			if stdout != "" || stderr == "" {
				t.Errorf("want empty stdout and an explanation, got stdout %q stderr %q", stdout, stderr)
			}
		})
	}
	t.Run("missing explicit config file", func(t *testing.T) {
		_, _, code := run("check", "--dsn", "sqlite://"+dbPath, "--config", filepath.Join(t.TempDir(), "nope.yml"))
		if code != cli.ExitUsage {
			t.Errorf("exit code = %d, want %d", code, cli.ExitUsage)
		}
	})
}

func TestCheckNamesUnmatchedExclusion(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")
	applyDDL(t, dbPath, "CREATE TABLE ok (id INTEGER PRIMARY KEY);")
	cfg := filepath.Join(t.TempDir(), "schemalens.yml")
	writeFile(t, cfg, "exclusions:\n  - rule: table-without-primary-key\n    object: main.gone\n    reason: legacy\n")
	_, stderr, code := run("check", "--dsn", "sqlite://"+dbPath, "--config", cfg)
	if code != cli.ExitFindings {
		t.Errorf("exit code = %d, want %d", code, cli.ExitFindings)
	}
	if !strings.Contains(stderr, "main.gone") || !strings.Contains(stderr, "table-without-primary-key") {
		t.Errorf("stderr should name the exclusion, got %q", stderr)
	}
}

func TestCheckFindsDefaultConfigInWorkingDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")
	applyDDL(t, dbPath, "CREATE TABLE audit_log (message TEXT);")
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "schemalens.yml"), "rules:\n  table-without-primary-key:\n    enabled: false\n")
	t.Chdir(dir)
	_, stderr, code := run("check", "--dsn", "sqlite://"+dbPath)
	if code != cli.ExitOK {
		t.Errorf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
}
