package text_test

import (
	"bytes"
	"testing"

	"github.com/mokevnin/schemalens/internal/finding"
	"github.com/mokevnin/schemalens/internal/text"
)

func render(t *testing.T, fs []finding.Finding, um []text.Unmatched) string {
	t.Helper()
	var buf bytes.Buffer
	if err := text.Render(&buf, fs, um); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestRenderClean(t *testing.T) {
	if got, want := render(t, nil, nil), "no findings\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderFindings(t *testing.T) {
	got := render(t, []finding.Finding{
		{Rule: "fk-without-index", Object: finding.Object{Schema: "main", Table: "posts", ForeignKey: "fk_user"},
			Message: "no index", Fix: "add one", Source: &finding.Position{File: "post.go", Line: 12}},
		{Rule: "table-without-primary-key", Object: finding.Object{Schema: "main", Table: "audit_log"},
			Message: "no pk", Fix: "add pk"},
	}, []text.Unmatched{{Rule: "narrow-key", Object: "main.t", Reason: "legacy"}})
	want := `fk-without-index  main.posts.fk_user
  no index
  fix: add one
  at: post.go:12

table-without-primary-key  main.audit_log
  no pk
  fix: add pk

exclusion matches nothing: rule narrow-key, object main.t (legacy)

2 findings, 1 stale exclusion
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
