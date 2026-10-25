// Package text renders findings for people reading a terminal.
package text

import (
	"fmt"
	"io"
	"strings"

	"github.com/mokevnin/schemalens/internal/finding"
)

// Unmatched is a configured exclusion that matched no finding.
type Unmatched struct {
	Rule, Object, Reason string
}

// Render writes one block per finding (rule, object, message, fix and the
// model source position when known), a stale-exclusion section when there are
// any, and a one-line summary. Findings keep their given order.
func Render(w io.Writer, findings []finding.Finding, unmatched []Unmatched) error {
	var b strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&b, "%s  %s\n", f.Rule, name(f.Object))
		fmt.Fprintf(&b, "  %s\n", f.Message)
		fmt.Fprintf(&b, "  fix: %s\n", f.Fix)
		if f.Source != nil {
			fmt.Fprintf(&b, "  at: %s:%d\n", f.Source.File, max(f.Source.Line, 1))
		}
		b.WriteByte('\n')
	}
	for _, u := range unmatched {
		fmt.Fprintf(&b, "exclusion matches nothing: rule %s, object %s (%s)\n", u.Rule, u.Object, u.Reason)
	}
	if len(unmatched) > 0 {
		b.WriteByte('\n')
	}
	switch {
	case len(findings) == 0 && len(unmatched) == 0:
		b.WriteString("no findings\n")
	default:
		fmt.Fprintf(&b, "%s, %s\n", plural(len(findings), "finding"), plural(len(unmatched), "stale exclusion"))
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// name is the dotted path of the database object, schema first.
func name(o finding.Object) string {
	parts := []string{o.Schema}
	for _, p := range []string{o.Table, o.Column, o.Index, o.ForeignKey, o.Trigger, o.View, o.Function} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ".")
}
