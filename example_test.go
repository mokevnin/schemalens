package schemalens_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/mokevnin/schemalens"
)

// exampleDB creates a SQLite database with ddl in a temporary directory and
// returns its DSN and a cleanup function.
func exampleDB(ddl string) (string, func()) {
	dir, err := os.MkdirTemp("", "schemalens-example")
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(dir, "app.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec(ddl); err != nil {
		log.Fatal(err)
	}
	_ = db.Close()
	return "sqlite://" + path, func() { _ = os.RemoveAll(dir) }
}

func ExampleCheck() {
	dsn, cleanup := exampleDB(`CREATE TABLE audit_log (message TEXT)`)
	defer cleanup()

	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn})
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range rep.Findings {
		fmt.Println(f.Rule, f.Object.Schema+"."+f.Object.Table)
	}
	fmt.Println("passed:", rep.Passed())
	// Output:
	// table-without-primary-key main.audit_log
	// passed: false
}

func ExampleCheck_exclusion() {
	dsn, cleanup := exampleDB(`CREATE TABLE audit_log (message TEXT)`)
	defer cleanup()

	cfg, err := schemalens.ParseConfig([]byte(`
exclusions:
  - rule: table-without-primary-key
    object: main.audit_log
    reason: append-only raw event log
  - rule: table-without-primary-key
    object: main.gone
    reason: table was dropped
`))
	if err != nil {
		log.Fatal(err)
	}
	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn, Config: cfg})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("findings:", len(rep.Findings))
	for _, e := range rep.UnmatchedExclusions {
		fmt.Println("stale:", e.Object)
	}
	fmt.Println("passed:", rep.Passed())
	// Output:
	// findings: 0
	// stale: main.gone
	// passed: false
}

func ExampleCheck_manifest() {
	dsn, cleanup := exampleDB(`CREATE TABLE users (id INTEGER NOT NULL PRIMARY KEY, email TEXT)`)
	defer cleanup()

	manifest := []byte(`{"version": 1, "tables": [{"name": "users", "columns": [
		{"name": "email", "required": true, "source": {"file": "user.go", "line": 18}}]}]}`)
	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn, Manifest: manifest})
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range rep.Findings {
		fmt.Printf("%s %s:%d\n", f.Rule, f.Source.File, f.Source.Line)
	}
	// Output:
	// required-field-nullable-column user.go:18
}

func ExampleReport_WriteJSON() {
	dsn, cleanup := exampleDB(`CREATE TABLE users (id INTEGER PRIMARY KEY)`)
	defer cleanup()

	rep, err := schemalens.Check(context.Background(), schemalens.Options{DSN: dsn})
	if err != nil {
		log.Fatal(err)
	}
	if err := rep.WriteJSON(os.Stdout); err != nil {
		log.Fatal(err)
	}
	// Output:
	// {
	//   "findings": []
	// }
}

func ExampleExplain() {
	e, _ := schemalens.Explain("table-without-primary-key")
	fmt.Println(e.ID, e.Dialects, e.DefaultOn)
	// Output:
	// table-without-primary-key [mysql postgres sqlite] true
}
