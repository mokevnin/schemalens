package gormadapter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mokevnin/schemalens/adapters/gorm/testdata/shop"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// migratedShop creates the sample project's schema with gorm's own AutoMigrate
// (naming strategy as in schemalens-gorm.yml) and returns the database path.
func migratedShop(t *testing.T) (string, *gorm.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gorm.sqlite")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{NamingStrategy: schema.NamingStrategy{TablePrefix: "app_"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&shop.User{}, &shop.Order{}, &shop.Company{}, &shop.Label{}, &shop.Language{}, &shop.Item{}); err != nil {
		t.Fatal(err)
	}
	return path, db
}

// checkShop runs `schemalens check --manifest` with the sample project's manifest.
func checkShop(t *testing.T, dsn string) (string, int) {
	t.Helper()
	manifest, stderr, code := runIn(t, "testdata/shop", bins.Adapter)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	work := t.TempDir()
	mpath := filepath.Join(work, "manifest.json")
	if err := os.WriteFile(mpath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(work, "schemalens.yml")
	// SQLite ignores VARCHAR(n), and gorm indexes no foreign key: those rules
	// would always fire, and they are not what these tests are about.
	rules := "rules:\n  max-length-not-enforced:\n    enabled: false\n  fk-without-index:\n    enabled: false\n"
	if err := os.WriteFile(cfg, []byte(rules), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runIn(t, work, bins.Schemalens, "check", "--dsn", dsn, "--config", cfg, "--manifest", mpath)
	if code == 2 {
		t.Fatalf("check failed: %s", stderr)
	}
	return stdout, code
}

func TestManyToManyJoinTablesCreatedByGormAreAccepted(t *testing.T) {
	path, _ := migratedShop(t)
	stdout, code := checkShop(t, "sqlite://"+path)
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, stdout)
	}
}

func TestManyToManyJoinTableMissingForeignKeyIsReportedAtTheField(t *testing.T) {
	path, db := migratedShop(t)
	// The join table exists with the right columns, but nothing ties
	// language_id to languages.
	if err := db.Exec(`DROP TABLE app_user_languages`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE app_user_languages (user_id integer NOT NULL, language_id integer NOT NULL, PRIMARY KEY (user_id, language_id),
		CONSTRAINT fk_users FOREIGN KEY (user_id) REFERENCES app_users(id))`).Error; err != nil {
		t.Fatal(err)
	}
	stdout, code := checkShop(t, "sqlite://"+path)
	if code != 1 {
		t.Fatalf("exit %d, want 1:\n%s", code, stdout)
	}
	for _, want := range []string{"relation-without-foreign-key", "app_user_languages", `"file": "models.go"`, `"line": 35`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("findings lack %q:\n%s", want, stdout)
		}
	}
}

func TestManyToManyJoinTableNameCanBeOverriddenInConfig(t *testing.T) {
	stdout, stderr, code := runIn(t, "testdata/joinoverride", bins.Adapter)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, `"name": "legacy_enrollments"`) || strings.Contains(stdout, `"name": "enrollments"`) {
		t.Errorf("join table not named by the override:\n%s", stdout)
	}
}
