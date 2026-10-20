package manifest_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mokevnin/schemalens/manifest"
)

func TestWritePrintsAVersionedManifestDocument(t *testing.T) {
	m := manifest.New()
	m.Tables = append(m.Tables, manifest.Table{
		Name:       "app_users",
		Columns:    []manifest.Column{{Name: "id", Required: true}, {Name: "email", MaxLen: 20}},
		PrimaryKey: []string{"id"},
	})
	var buf bytes.Buffer
	if err := m.Write(&buf); err != nil {
		t.Fatal(err)
	}
	want := `{"version":1,"tables":[{"name":"app_users","columns":[{"name":"id","required":true},{"name":"email","max_length":20}],"primary_key":["id"]}]}`
	if got := compact(buf.String()); got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestAnEmptyManifestHasAnEmptyTableListNotNull(t *testing.T) {
	var buf bytes.Buffer
	if err := (&manifest.Manifest{Version: manifest.Version}).Write(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"tables": []`) {
		t.Errorf("tables should be [], got %s", buf.String())
	}
}

func compact(s string) string {
	var b strings.Builder
	inStr := false
	for _, r := range s {
		if r == '"' {
			inStr = !inStr
		}
		if !inStr && (r == ' ' || r == '\n' || r == '\t') {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
