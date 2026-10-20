package ruledocs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mokevnin/schemalens/internal/ruledocs"
	"github.com/mokevnin/schemalens/internal/rules"
)

const docsDir = "../../docs/rules"

// A docs page must exist for every registered rule and match what the
// generator renders from the registry. Run `mise run docs` to regenerate.
func TestRuleDocsMatchRegistry(t *testing.T) {
	want, err := ruledocs.Render()
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range want {
		got, err := os.ReadFile(filepath.Join(docsDir, name))
		if err != nil {
			t.Errorf("docs/rules/%s is missing; run `mise run docs`", name)
			continue
		}
		if string(got) != content {
			t.Errorf("docs/rules/%s is out of date; run `mise run docs`", name)
		}
	}
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if _, ok := want[e.Name()]; !ok {
			t.Errorf("docs/rules/%s belongs to no registered rule; run `mise run docs`", e.Name())
		}
	}
}

func TestRenderedPageCarriesTheExplainText(t *testing.T) {
	pages, err := ruledocs.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rules.All() {
		page := pages[r.ID+".md"]
		for _, text := range []string{r.ID, r.Explain.Why, r.Explain.Fix, r.Explain.Exception} {
			if !strings.Contains(page, text) {
				t.Errorf("page for %s lacks %q", r.ID, text)
			}
		}
	}
}
