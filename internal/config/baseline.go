package config

import (
	"bytes"

	"gopkg.in/yaml.v3"

	"github.com/mokevnin/schemalens/internal/finding"
)

// BaselineReason is the reason recorded on exclusions written by baseline.
const BaselineReason = "baseline"

// FormatObject is the inverse of ParseObject.
func FormatObject(o finding.Object) string {
	s := o.Schema + "." + o.Table
	switch {
	case o.Column != "":
		s += ".column:" + o.Column
	case o.Index != "":
		s += ".index:" + o.Index
	case o.ForeignKey != "":
		s += ".foreign_key:" + o.ForeignKey
	}
	return s
}

// AddBaseline appends an exclusion with reason "baseline" to the YAML in data
// for each finding not already excluded, editing the document in place so
// existing exclusions, reasons and comments survive. data may be empty. It
// returns the new content and how many exclusions were added; when that is
// zero the content is data unchanged. Findings that data already excludes
// (see Apply) are skipped, so running it twice adds nothing.
func AddBaseline(data []byte, findings []finding.Finding) ([]byte, int, error) {
	cfg, err := Parse(data)
	if err != nil {
		return nil, 0, err
	}
	kept, _ := cfg.Apply(findings)
	seen := map[Exclusion]bool{}
	var added []Exclusion
	for _, f := range kept {
		e := Exclusion{Rule: f.Rule, Object: FormatObject(f.Object), Reason: BaselineReason}
		if !seen[e] {
			seen[e] = true
			added = append(added, e)
		}
	}
	if len(added) == 0 {
		return data, 0, nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, 0, err
	}
	if doc.Kind == 0 { // empty input
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode { // a null document, e.g. only comments
		root.Kind, root.Tag, root.Value, root.Content = yaml.MappingNode, "!!map", "", nil
	}
	var list *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "exclusions" {
			list = root.Content[i+1]
		}
	}
	if list == nil {
		list = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "exclusions"}, list)
	}
	if list.Kind != yaml.SequenceNode { // `exclusions:` with no value
		list.Kind, list.Tag, list.Value = yaml.SequenceNode, "!!seq", ""
	}
	list.Style = 0
	for _, e := range added {
		var n yaml.Node
		if err := n.Encode(e); err != nil {
			return nil, 0, err
		}
		list.Content = append(list.Content, &n)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, 0, err
	}
	if err := enc.Close(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), len(added), nil
}
