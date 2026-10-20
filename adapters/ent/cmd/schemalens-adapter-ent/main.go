// Command schemalens-adapter-ent prints the manifest of an ent schema package.
package main

import (
	"os"

	entadapter "github.com/mokevnin/schemalens/adapters/ent"
)

func main() {
	os.Exit(entadapter.Run(os.Args[1:], os.Stdout, os.Stderr))
}
