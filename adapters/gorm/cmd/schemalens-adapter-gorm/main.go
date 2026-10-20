// Command schemalens-adapter-gorm prints the manifest of a gorm project's models.
package main

import (
	"os"

	gormadapter "github.com/mokevnin/schemalens/adapters/gorm"
)

func main() {
	os.Exit(gormadapter.Run(os.Args[1:], os.Stdout, os.Stderr))
}
