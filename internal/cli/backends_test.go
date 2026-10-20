package cli_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"sync"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	_ "modernc.org/sqlite"

	"github.com/mokevnin/schemalens/internal/dialect"
)

// backend creates a fresh, empty database for one golden case, applies the
// DDL and returns the DSN `schemalens check` should inspect.
type backend struct {
	dialect dialect.Dialect
	setup   func(t *testing.T, ddl string) string
}

var backends = []backend{
	{dialect.SQLite, setupSQLite},
	{dialect.Postgres, setupPostgres},
	{dialect.MySQL, setupMySQL},
}

// testDatabase is the database every container case runs in. MySQL reports
// it as the schema name, so it is fixed and shows up in golden files.
const testDatabase = "schemalens"

func setupSQLite(t *testing.T, ddl string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "db.sqlite")
	applyDDL(t, path, ddl)
	return "sqlite://" + path
}

// applyDDL runs ddl against the SQLite database at path, creating it if needed.
func applyDDL(t *testing.T, path, ddl string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("applying DDL: %v", err)
	}
}

// server is one container shared by every case of a test run. Testcontainers'
// reaper removes it when the test process exits.
type server struct {
	once sync.Once
	host string // host:port
	err  error
}

var (
	pgServer    server
	mysqlServer server
)

// skipWithoutDocker skips the test with a clear message when no container
// runtime is reachable, so the suite stays runnable on machines without Docker.
func skipWithoutDocker(t *testing.T) {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)
}

func setupPostgres(t *testing.T, ddl string) string {
	t.Helper()
	skipWithoutDocker(t)
	pgServer.once.Do(func() {
		ctx := context.Background()
		c, err := tcpostgres.Run(ctx, "postgres:17-alpine",
			tcpostgres.WithUsername("postgres"), tcpostgres.WithPassword("secret"),
			tcpostgres.BasicWaitStrategies())
		if err != nil {
			pgServer.err = err
			return
		}
		pgServer.host, pgServer.err = c.PortEndpoint(ctx, "5432/tcp", "")
	})
	if pgServer.err != nil {
		t.Fatalf("postgres container: %v", pgServer.err)
	}
	dsn := func(db string) string {
		return fmt.Sprintf("postgres://postgres:secret@%s/%s?sslmode=disable", pgServer.host, db)
	}
	recreate(t, "pgx", dsn("postgres"), dsn(testDatabase), ddl)
	return dsn(testDatabase)
}

func setupMySQL(t *testing.T, ddl string) string {
	t.Helper()
	skipWithoutDocker(t)
	mysqlServer.once.Do(func() {
		ctx := context.Background()
		c, err := tcmysql.Run(ctx, "mysql:8.4",
			tcmysql.WithUsername("root"), tcmysql.WithPassword("secret"))
		if err != nil {
			mysqlServer.err = err
			return
		}
		mysqlServer.host, mysqlServer.err = c.PortEndpoint(ctx, "3306/tcp", "")
	})
	if mysqlServer.err != nil {
		t.Fatalf("mysql container: %v", mysqlServer.err)
	}
	// multiStatements lets the DDL file hold several statements.
	driver := func(db string) string {
		return fmt.Sprintf("root:secret@tcp(%s)/%s?multiStatements=true", mysqlServer.host, db)
	}
	recreate(t, "mysql", driver(""), driver(testDatabase), ddl)
	u := url.URL{Scheme: "mysql", User: url.UserPassword("root", "secret"), Host: mysqlServer.host, Path: "/" + testDatabase}
	return u.String()
}

// recreate drops and recreates the test database through adminDSN, then
// applies ddl through dbDSN.
func recreate(t *testing.T, driver, adminDSN, dbDSN, ddl string) {
	t.Helper()
	ctx := context.Background()
	admin, err := sql.Open(driver, adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = admin.Close() }()
	for _, q := range []string{
		"DROP DATABASE IF EXISTS " + testDatabase,
		"CREATE DATABASE " + testDatabase,
	} {
		if _, err := admin.ExecContext(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	db, err := sql.Open(driver, dbDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		t.Fatalf("applying DDL: %v", err)
	}
}
