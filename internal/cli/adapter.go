package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/mokevnin/schemalens/internal/manifest"
)

// adapterPrefix is the executable name prefix of an adapter: `--adapter ent`
// runs `schemalens-adapter-ent`. See docs/adapters.md.
const adapterPrefix = "schemalens-adapter-"

// stringList is a repeatable string flag.
type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, " ") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

// runAdapter finds schemalens-adapter-<name> on PATH, runs it in the current
// directory with args and returns the manifest it prints, validated exactly
// like a manifest file.
func runAdapter(name string, args []string, stderr io.Writer) (*manifest.Manifest, int) {
	exe := adapterPrefix + name
	path, err := exec.LookPath(exe)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens check: adapter %q: executable %s not found (searched PATH: %s)\n", name, exe, os.Getenv("PATH"))
		return nil, ExitUsage
	}
	var out, errb bytes.Buffer
	cmd := exec.Command(path, args...) //nolint:gosec // running the adapter the user asked for is the feature
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			_, _ = fmt.Fprintf(stderr, "schemalens check: adapter %s failed: %v\n", exe, err)
		} else {
			_, _ = fmt.Fprintf(stderr, "schemalens check: adapter %s: %v\n", exe, err)
		}
		if msg := strings.TrimSpace(errb.String()); msg != "" {
			_, _ = fmt.Fprintf(stderr, "%s\n", msg)
		}
		return nil, ExitUsage
	}
	m, err := manifest.Parse(out.Bytes())
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens check: manifest from adapter %s: %v\n", exe, err)
		return nil, ExitUsage
	}
	return m, ExitOK
}
