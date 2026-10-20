// Package entadapter is the schemalens adapter for ent: it loads an ent
// schema package and prints the model manifest that `schemalens check
// --adapter ent` compares with the database.
package entadapter

import (
	"encoding/json"
	"fmt"
	"io"
)

// DefaultSchemaPath is where ent projects keep their schema by convention.
const DefaultSchemaPath = "./ent/schema"

// Run is the adapter's entry point. args are the adapter arguments schemalens
// passes through (--adapter-arg): at most one, the schema package directory,
// relative to the working directory. It prints the manifest to stdout and
// returns the exit code; failures go to stderr and leave stdout empty.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		_, _ = fmt.Fprintf(stderr, "usage: schemalens-adapter-ent [schema-package-dir]  (default %s)\n", DefaultSchemaPath)
		return 2
	}
	path := DefaultSchemaPath
	if len(args) == 1 {
		path = args[0]
	}
	m, err := loadManifest(path)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	_, _ = stdout.Write(append(data, '\n'))
	return 0
}
