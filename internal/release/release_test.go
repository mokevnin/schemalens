// Package release holds checks on the release pipeline configuration. The pipeline itself
// (release-please, goreleaser) cannot run in tests; these tests pin what the ticket promises.
package release

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func repoFile(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type goreleaser struct {
	Builds []struct {
		ID     string   `yaml:"id"`
		Main   string   `yaml:"main"`
		Binary string   `yaml:"binary"`
		Goos   []string `yaml:"goos"`
		Goarch []string `yaml:"goarch"`
	} `yaml:"builds"`
	Archives []struct {
		ID     string   `yaml:"id"`
		Builds []string `yaml:"ids"`
	} `yaml:"archives"`
	Changelog struct {
		Use    string `yaml:"use"`
		Groups []struct {
			Title string `yaml:"title"`
			Regex string `yaml:"regexp"`
		} `yaml:"groups"`
	} `yaml:"changelog"`
}

func loadGoreleaser(t *testing.T) goreleaser {
	t.Helper()
	var g goreleaser
	if err := yaml.Unmarshal(repoFile(t, ".goreleaser.yaml"), &g); err != nil {
		t.Fatal(err)
	}
	return g
}

func TestGoreleaserBuildsCoreAndEachBuiltInAdapterSeparately(t *testing.T) {
	g := loadGoreleaser(t)
	want := map[string]string{
		"schemalens":              "./cmd/schemalens",
		"schemalens-adapter-ent":  "./cmd/schemalens-adapter-ent",
		"schemalens-adapter-gorm": "./cmd/schemalens-adapter-gorm",
		"schemalens-adapter-bun":  "./cmd/schemalens-adapter-bun",
	}
	for binary, main := range want {
		found := false
		for _, b := range g.Builds {
			if b.Binary != binary {
				continue
			}
			found = true
			if b.Main != main {
				t.Errorf("%s: main = %q, want %q", binary, b.Main, main)
			}
			for _, goos := range []string{"linux", "darwin", "windows"} {
				if !slices.Contains(b.Goos, goos) {
					t.Errorf("%s: missing goos %s", binary, goos)
				}
			}
			for _, arch := range []string{"amd64", "arm64"} {
				if !slices.Contains(b.Goarch, arch) {
					t.Errorf("%s: missing goarch %s", binary, arch)
				}
			}
			archived := 0
			for _, a := range g.Archives {
				if slices.Contains(a.Builds, b.ID) {
					archived++
					if len(a.Builds) != 1 {
						t.Errorf("%s: archive %s mixes builds %v", binary, a.ID, a.Builds)
					}
				}
			}
			if archived != 1 {
				t.Errorf("%s: in %d archives, want exactly 1", binary, archived)
			}
		}
		if !found {
			t.Errorf("no build for %s", binary)
		}
	}
	if len(g.Builds) != len(want) {
		t.Errorf("%d builds, want %d (libraries are not released as binaries)", len(g.Builds), len(want))
	}
}

func TestGoreleaserChangelogComesFromCommitsGroupedByType(t *testing.T) {
	g := loadGoreleaser(t)
	if g.Changelog.Use != "git" {
		t.Errorf("changelog.use = %q, want git", g.Changelog.Use)
	}
	if len(g.Changelog.Groups) < 3 {
		t.Errorf("want changelog groups by Conventional Commit type, got %d", len(g.Changelog.Groups))
	}
}

func TestReleasePleaseProducesOneVersionForTheRepo(t *testing.T) {
	var cfg struct {
		Packages map[string]json.RawMessage `json:"packages"`
	}
	if err := json.Unmarshal(repoFile(t, "release-please-config.json"), &cfg); err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Packages["."]; !ok || len(cfg.Packages) != 1 {
		t.Errorf("packages = %v, want only the root package \".\"", cfg.Packages)
	}
	var manifest map[string]string
	if err := json.Unmarshal(repoFile(t, ".release-please-manifest.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	if _, ok := manifest["."]; !ok || len(manifest) != 1 {
		t.Errorf("manifest = %v, want only \".\"", manifest)
	}
}

func TestReleaseWorkflowsRunReleasePleaseAndGoreleaser(t *testing.T) {
	rp := string(repoFile(t, ".github/workflows/release-please.yml"))
	if !strings.Contains(rp, "release-please-action") {
		t.Error("release-please workflow does not use release-please-action")
	}
	rel := string(repoFile(t, ".github/workflows/release.yml"))
	if !strings.Contains(rel, "goreleaser release") {
		t.Error("release workflow does not run goreleaser")
	}
}

func TestCoreModuleDoesNotRequireAnyAdapter(t *testing.T) {
	for _, f := range []string{"go.mod", "go.sum"} {
		if strings.Contains(string(repoFile(t, f)), "schemalens/adapters") {
			t.Errorf("core %s mentions an adapter module", f)
		}
	}
}
