package gormext_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gorm.io/gorm/schema"

	"github.com/mokevnin/schemalens/adapters/gormext"
)

type Course struct {
	ID    uint
	Title string
}

type Student struct {
	ID      uint
	Courses []Course `gorm:"many2many:enrollments"`
}

func TestManyToManyJoinTableIsDeclaredAndAccepted(t *testing.T) {
	namer := schema.NamingStrategy{TablePrefix: "uni_"}
	dbPath, _ := migrate(t, namer, &Student{}, &Course{})
	m, err := gormext.Manifest(namer, &Student{}, &Course{})
	if err != nil {
		t.Fatal(err)
	}
	join := tableNamed(t, m, "uni_enrollments")
	if len(join.JoinTableFor) != 2 {
		t.Errorf("join_table_for = %+v, want the two joined tables", join.JoinTableFor)
	}
	var doc bytes.Buffer
	if err := m.Write(&doc); err != nil {
		t.Fatal(err)
	}
	mpath := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(mpath, doc.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	out, errOut, code := runCheck(t, "--dsn", "sqlite://"+dbPath, "--manifest", mpath)
	if code != 0 {
		t.Fatalf("check exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
}
