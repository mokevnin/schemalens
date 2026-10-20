package bunadapter

import (
	"strings"
)

// tag is a parsed `bun:"..."` tag. bun's own parser lives in an internal
// package, so the grammar is reproduced here (github.com/uptrace/bun,
// internal/tagparser, BSD-2-Clause):
//
//	name,key,key:value,key:"quoted, value",key:func(a, b),key:v1,key:v2
//
// The first item without a colon is the column name (the table name on
// bun.BaseModel); a repeated key collects all its values; parentheses and
// double quotes protect commas in values.
type tag struct {
	Name    string
	Options map[string][]string
}

func (t tag) has(name string) bool {
	_, ok := t.Options[name]
	return ok
}

// option returns the last value given for name.
func (t tag) option(name string) (string, bool) {
	if vs, ok := t.Options[name]; ok {
		return vs[len(vs)-1], true
	}
	return "", false
}

func parseTag(s string) tag {
	p := tagParser{s: s}
	for p.valid() {
		p.keyValue()
		if p.peek() == ',' {
			p.i++
		}
	}
	return p.tag
}

type tagParser struct {
	s        string
	i        int
	tag      tag
	seenName bool
}

func (p *tagParser) setName(name string) {
	if p.seenName {
		p.add(name, "")
		return
	}
	p.seenName = true
	p.tag.Name = name
}

func (p *tagParser) add(key, value string) {
	p.seenName = true
	if key == "" {
		return
	}
	if p.tag.Options == nil {
		p.tag.Options = map[string][]string{}
	}
	p.tag.Options[key] = append(p.tag.Options[key], value)
}

func (p *tagParser) keyValue() {
	start := p.i
	for p.valid() {
		switch c := p.read(); c {
		case ',':
			p.setName(p.s[start : p.i-1])
			return
		case ':':
			key := p.s[start : p.i-1]
			p.add(key, p.value())
			return
		case '"':
			p.setName(p.quoted())
			return
		}
	}
	p.setName(p.s[start:p.i])
}

func (p *tagParser) value() string {
	start := p.i
	for p.valid() {
		switch c := p.read(); c {
		case '"':
			return p.quoted()
		case ',':
			return p.s[start : p.i-1]
		case '(':
			p.skipPairs()
		}
	}
	return p.s[start:p.i]
}

func (p *tagParser) quoted() string {
	var b strings.Builder
	for p.valid() {
		switch c := p.read(); c {
		case '\\':
			b.WriteByte(p.read())
		case '"':
			return b.String()
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// skipPairs consumes input up to the parenthesis closing one just opened.
func (p *tagParser) skipPairs() {
	depth := 0
	for p.valid() {
		switch c := p.read(); c {
		case '"':
			_ = p.quoted()
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return
			}
			depth--
		}
	}
}

func (p *tagParser) valid() bool { return p.i < len(p.s) }

func (p *tagParser) read() byte {
	if !p.valid() {
		return 0
	}
	c := p.s[p.i]
	p.i++
	return c
}

func (p *tagParser) peek() byte {
	if !p.valid() {
		return 0
	}
	return p.s[p.i]
}
