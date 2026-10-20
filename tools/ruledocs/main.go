// Command ruledocs regenerates docs/rules from the rule registry.
package main

import (
	"fmt"
	"os"

	"github.com/mokevnin/schemalens/internal/ruledocs"
)

func main() {
	dir := "docs/rules"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if err := ruledocs.Write(dir); err != nil {
		fmt.Fprintln(os.Stderr, "ruledocs:", err)
		os.Exit(1)
	}
}
