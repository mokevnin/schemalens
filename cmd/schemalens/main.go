// Command schemalens checks a live database against rules and prints findings.
package main

import (
	"os"

	"github.com/mokevnin/schemalens/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
