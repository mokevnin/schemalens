package entadapter_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	entadapter "github.com/mokevnin/schemalens/adapters/ent"
)

var update = flag.Bool("update", false, "rewrite golden files")

const sampleSchema = "testdata/sample/ent/schema"

func printManifest(t *testing.T, args ...string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := entadapter.Run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("adapter exit code = %d, stderr: %s", code, stderr.String())
	}
	return stdout.String()
}

func TestSampleProjectManifestMatchesGolden(t *testing.T) {
	got := printManifest(t, sampleSchema)
	golden := filepath.Join("testdata", "sample", "manifest.golden.json")
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil { //nolint:gosec // golden file
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("manifest differs from %s (run with -update and review the diff)\n--- got ---\n%s", golden, got)
	}
}

func TestMissingSchemaPackageFailsWithMessage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := entadapter.Run([]string{"testdata/nope"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout must stay empty on failure, got %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Error("stderr is empty, want the cause")
	}
}
