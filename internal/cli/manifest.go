package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/mokevnin/schemalens"
)

// runManifestSchema prints the JSON Schema of the manifest.
func runManifestSchema(stdout, stderr io.Writer) int {
	if _, err := stdout.Write(schemalens.ManifestSchema()); err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens manifest-schema: write output: %v\n", err)
		return ExitUsage
	}
	return ExitOK
}

// loadManifest reads and validates the manifest at path. An empty path means
// no manifest: it returns nil and ExitOK.
func loadManifest(path string, stderr io.Writer) (*schemalens.Manifest, int) {
	if path == "" {
		return nil, ExitOK
	}
	data, err := os.ReadFile(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens check: read manifest: %v\n", err)
		return nil, ExitUsage
	}
	m, err := schemalens.ParseManifest(data)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens check: manifest %s: %v\n", path, err)
		return nil, ExitUsage
	}
	return m, ExitOK
}
