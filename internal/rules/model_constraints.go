package rules

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// Helpers shared by the rules that ask whether the database enforces a
// promise of the model: uniqueness, string length, enum values.

// nameSet returns the distinct names, sorted, for order-insensitive
// comparison of column lists.
func nameSet(names []string) []string {
	out := slices.Clone(names)
	slices.Sort(out)
	return slices.Compact(out)
}

// quoteList renders names as `"a", "b"`.
func quoteList(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = strconv.Quote(n)
	}
	return strings.Join(q, ", ")
}

// tableChecks returns the CHECK constraints of a table.
func tableChecks(t *schema.Table) []*schema.Check {
	var out []*schema.Check
	for _, a := range t.Attrs {
		if c, ok := a.(*schema.Check); ok {
			out = append(out, c)
		}
	}
	return out
}

var (
	sqlLiteral = regexp.MustCompile(`'(?:[^']|'')*'`)
	// Casts PostgreSQL adds when it renders a stored CHECK expression.
	sqlCast = regexp.MustCompile(`::(?:character varying|character|bpchar|varchar|text|citext|name|integer|bigint|smallint|int4|int8)(?:\[\])?`)
	// MySQL prefixes literals with their character set: _utf8mb4'abc'.
	sqlCharset = regexp.MustCompile(`_[a-z0-9]+\?`)
	sqlPunct   = strings.NewReplacer("(", " ", ")", " ", "[", " ", "]", " ", ",", " ", "`", "", `"`, "")
	sqlSpaces  = regexp.MustCompile(`\s+`)
)

// skeleton flattens a CHECK expression so engines' different renderings of
// the same constraint compare equal: string literals become "?" (their values
// are returned in order), casts, quoting, parentheses and charset prefixes
// disappear, and the rest is lower-cased and space separated.
func skeleton(expr string) (string, []string) {
	var vals []string
	s := sqlLiteral.ReplaceAllStringFunc(expr, func(m string) string {
		vals = append(vals, strings.ReplaceAll(m[1:len(m)-1], "''", "'"))
		return "?"
	})
	s = sqlCast.ReplaceAllString(s, "")
	s = sqlCharset.ReplaceAllString(s, "?")
	s = sqlPunct.Replace(s)
	s = strings.ToLower(sqlSpaces.ReplaceAllString(s, " "))
	return strings.TrimSpace(s), vals
}

// stripNullEscape removes a leading "col is null or": a CHECK that lets null
// through still enforces the value constraint on every stored value.
func stripNullEscape(sk, col string) string {
	return strings.TrimPrefix(sk, col+" is null or ")
}

// checkMaxLength returns the length limit a CHECK puts on col, if the
// expression is a length comparison of that column (possibly one conjunct of
// an AND).
func checkMaxLength(expr, col string) (int, bool) {
	sk, _ := skeleton(expr)
	c := strings.ToLower(col)
	sk = stripNullEscape(sk, c)
	if strings.Contains(sk, " or ") {
		return 0, false
	}
	re := regexp.MustCompile(`(?:^| )(?:char_length|character_length|length) ` + regexp.QuoteMeta(c) + ` (<=|<) (\d+)(?: |$)`)
	best, found := 0, false
	for _, m := range re.FindAllStringSubmatch(sk, -1) {
		n, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		if m[1] == "<" {
			n--
		}
		if !found || n < best {
			best, found = n, true
		}
	}
	return best, found
}

// checkEnumValues returns the value list of a CHECK that restricts col to a
// list of literals: col IN (...), col = ANY (ARRAY[...]) or col = 'a' OR col =
// 'b'. The whole expression must be that restriction.
func checkEnumValues(expr, col string) ([]string, bool) {
	sk, vals := skeleton(expr)
	c := regexp.QuoteMeta(strings.ToLower(col))
	sk = stripNullEscape(sk, strings.ToLower(col))
	list := regexp.MustCompile(`^` + c + ` (?:in|= any(?: array)?) \?(?: \?)*$`)
	ors := regexp.MustCompile(`^` + c + ` = \?(?: or ` + c + ` = \?)*$`)
	if len(vals) == 0 || (!list.MatchString(sk) && !ors.MatchString(sk)) {
		return nil, false
	}
	return vals, true
}

// describeSet renders a value list for messages.
func describeSet(vals []string) string {
	return fmt.Sprintf("(%s)", quoteList(nameSet(vals)))
}
