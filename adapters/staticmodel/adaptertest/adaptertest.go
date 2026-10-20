// Package adaptertest holds the test plumbing the static adapters share:
// building the adapter and schemalens binaries, running them in a sample
// project and comparing output with golden files. Adapter tests drive the
// built binaries, never internal functions.
package adaptertest

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Binaries are the executables a test run uses.
type Binaries struct {
	Adapter    string // path of schemalens-adapter-<name>
	Schemalens string // path of schemalens
	dir        string
}

// Build compiles the adapter package (an import path, built within the
// workspace) and the schemalens command into a temporary directory. Call it
// from TestMain and Close the result after m.Run.
func Build(name, adapterPkg string) (*Binaries, error) {
	dir, err := os.MkdirTemp("", name+"-adapter-test")
	if err != nil {
		return nil, err
	}
	b := &Binaries{
		Adapter:    filepath.Join(dir, "schemalens-adapter-"+name),
		Schemalens: filepath.Join(dir, "schemalens"),
		dir:        dir,
	}
	for out, pkg := range map[string]string{
		b.Adapter:    adapterPkg,
		b.Schemalens: "github.com/mokevnin/schemalens/cmd/schemalens",
	} {
		if msg, err := exec.Command("go", "build", "-o", out, pkg).CombinedOutput(); err != nil {
			_ = os.RemoveAll(dir)
			return nil, errors.New(string(msg))
		}
	}
	return b, nil
}

// Close removes the binaries.
func (b *Binaries) Close() { _ = os.RemoveAll(b.dir) }

// PathEnv is the environment with the adapter binary's directory on PATH, so
// `schemalens check --adapter <name>` finds it.
func (b *Binaries) PathEnv() []string {
	return append(os.Environ(), "PATH="+filepath.Dir(b.Adapter)+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// RunIn runs bin in dir and returns stdout, stderr and the exit code.
func RunIn(t *testing.T, env []string, dir, bin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = env
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return out.String(), errb.String(), code
}

// Golden compares got with the file at path, or rewrites the file when update
// is set.
func Golden(t *testing.T, update bool, path, got string) {
	t.Helper()
	if update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil { //nolint:gosec // golden file
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Errorf("%s differs from adapter output (rerun with -update to review):\n%s", path, got)
	}
}
