package cli_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/mokevnin/schemalens/internal/cli"
)

type ruleEntry struct {
	ID       string   `json:"id"`
	Dialects []string `json:"dialects"`
}

func listRules(t *testing.T) []ruleEntry {
	t.Helper()
	stdout, stderr, code := run("rules", "--format", "json")
	if code != cli.ExitOK {
		t.Fatalf("rules --format json: exit %d, stderr: %s", code, stderr)
	}
	var doc struct {
		Rules []ruleEntry `json:"rules"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("rules JSON is invalid: %v\n%s", err, stdout)
	}
	return doc.Rules
}

func TestRulesJSONIsSortedAndListsDialects(t *testing.T) {
	rules := listRules(t)
	if len(rules) == 0 {
		t.Fatal("no rules listed")
	}
	ids := make([]string, len(rules))
	for i, r := range rules {
		ids[i] = r.ID
		if len(r.Dialects) == 0 || !slices.IsSorted(r.Dialects) {
			t.Errorf("rule %s: dialects %v must be non-empty and sorted", r.ID, r.Dialects)
		}
	}
	if !slices.IsSorted(ids) {
		t.Errorf("rules are not sorted by id: %v", ids)
	}
	if !slices.Contains(ids, "table-without-primary-key") {
		t.Errorf("table-without-primary-key missing from %v", ids)
	}
}

func TestRulesTextListsIDAndDialects(t *testing.T) {
	stdout, _, code := run("rules")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "table-without-primary-key") || !strings.Contains(stdout, "sqlite") {
		t.Errorf("output lacks rule id or dialect:\n%s", stdout)
	}
	again, _, _ := run("rules")
	if again != stdout {
		t.Error("two runs differ")
	}
}

func TestRulesRejectsUnknownFormat(t *testing.T) {
	_, stderr, code := run("rules", "--format", "xml")
	if code != cli.ExitUsage || stderr == "" {
		t.Errorf("exit %d, stderr %q; want usage error", code, stderr)
	}
}

// Every registered rule must be explainable with all three sections filled in.
// Adding a rule without an explanation fails here.
func TestEveryRuleHasAnExplanation(t *testing.T) {
	for _, r := range listRules(t) {
		t.Run(r.ID, func(t *testing.T) {
			stdout, stderr, code := run("explain", r.ID)
			if code != cli.ExitOK {
				t.Fatalf("exit %d, stderr: %s", code, stderr)
			}
			for _, section := range []string{"Why", "Fix", "Exception"} {
				if strings.TrimSpace(sectionBody(stdout, section)) == "" {
					t.Errorf("section %q is empty in:\n%s", section, stdout)
				}
			}
		})
	}
}

func TestExplainShowsRuleText(t *testing.T) {
	stdout, _, code := run("explain", "table-without-primary-key")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.HasPrefix(stdout, "table-without-primary-key\n") {
		t.Errorf("output should start with the rule id:\n%s", stdout)
	}
	if !strings.Contains(sectionBody(stdout, "Why"), "primary key") {
		t.Errorf("Why section lacks substance:\n%s", stdout)
	}
}

func TestExplainUsageErrors(t *testing.T) {
	tests := map[string][]string{
		"unknown rule": {"explain", "no-such-rule"},
		"no rule id":   {"explain"},
		"extra args":   {"explain", "a", "b"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, code := run(args...)
			if code != cli.ExitUsage {
				t.Errorf("exit %d, want %d", code, cli.ExitUsage)
			}
			if stdout != "" || stderr == "" {
				t.Errorf("stdout %q stderr %q", stdout, stderr)
			}
		})
	}
}

// sectionBody returns the text under a "<name>:" heading up to the next heading.
func sectionBody(out, name string) string {
	var body []string
	in := false
	for line := range strings.SplitSeq(out, "\n") {
		if line != "" && !strings.HasPrefix(line, " ") && strings.HasSuffix(line, ":") {
			in = line == name+":"
			continue
		}
		if in {
			body = append(body, line)
		}
	}
	return strings.Join(body, "\n")
}
