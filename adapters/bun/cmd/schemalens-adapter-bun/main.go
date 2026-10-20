// Command schemalens-adapter-bun prints the manifest of a bun project's models.
package main

import (
	"os"

	bunadapter "github.com/mokevnin/schemalens/adapters/bun"
)

func main() {
	os.Exit(bunadapter.Run(os.Args[1:], os.Stdout, os.Stderr))
}
