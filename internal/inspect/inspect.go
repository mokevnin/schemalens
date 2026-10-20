// Package inspect reads a live database into Atlas's schema model.
package inspect

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite" // registers the "sqlite" driver

	"github.com/mokevnin/schemalens/internal/dialect"
)

// Result is an inspected database.
type Result struct {
	Dialect dialect.Dialect
	Realm   *schema.Realm
	// DB stays open for rules that query the catalog directly; Close it.
	DB *sql.DB
}

// Close releases the database handle.
func (r *Result) Close() { _ = r.DB.Close() }

// Inspect connects to the database named by dsn and inspects it. The DSN
// scheme selects the dialect: sqlite://<path>, postgres://user:pass@host/db or
// mysql://user:pass@host/db. Only the schema the DSN points at is inspected:
// the connection's current schema for PostgreSQL ("public" unless the DSN sets
// search_path), the database for MySQL. SQLite files are opened read-only and
// never created.
func Inspect(ctx context.Context, dsn string) (*Result, error) {
	scheme, rest, ok := strings.Cut(dsn, "://")
	if !ok {
		return nil, fmt.Errorf("invalid dsn %q: expected scheme://location", dsn)
	}
	switch scheme {
	case "sqlite":
		return inspectSQLite(ctx, rest)
	case "postgres", "postgresql":
		return inspectPostgres(ctx, dsn)
	case "mysql":
		return inspectMySQL(ctx, dsn)
	default:
		return nil, fmt.Errorf("unsupported dsn scheme %q (supported: sqlite, postgres, mysql)", scheme)
	}
}

func inspectSQLite(ctx context.Context, path string) (*Result, error) {
	if path == "" {
		return nil, fmt.Errorf("invalid dsn: sqlite path is empty")
	}
	u := url.URL{Scheme: "file", Opaque: escapePath(path), RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, fmt.Errorf("open sqlite database %q: %w", path, err)
	}
	ok := false
	defer func() {
		if !ok {
			_ = db.Close()
		}
	}()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("open sqlite database %q: %w", path, err)
	}
	drv, err := sqlite.Open(db)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database %q: %w", path, err)
	}
	realm, err := drv.InspectRealm(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("inspect sqlite database %q: %w", path, err)
	}
	ok = true
	return &Result{Dialect: dialect.SQLite, Realm: realm, DB: db}, nil
}

func inspectPostgres(ctx context.Context, dsn string) (*Result, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid postgres dsn: %w", err)
	}
	// schemalens only reads; make the server refuse writes on this session.
	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = map[string]string{}
	}
	cfg.RuntimeParams["default_transaction_read_only"] = "on"
	db := stdlib.OpenDB(*cfg)
	ok := false
	defer func() {
		if !ok {
			_ = db.Close()
		}
	}()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connect to postgres database %q: %w", cfg.Database, err)
	}
	var current string
	if err := db.QueryRowContext(ctx, "SELECT current_schema()").Scan(&current); err != nil {
		return nil, fmt.Errorf("read current postgres schema: %w", err)
	}
	drv, err := postgres.Open(db)
	if err != nil {
		return nil, fmt.Errorf("open postgres database %q: %w", cfg.Database, err)
	}
	realm, err := drv.InspectRealm(ctx, &schema.InspectRealmOption{Schemas: []string{current}})
	if err != nil {
		return nil, fmt.Errorf("inspect postgres database %q: %w", cfg.Database, err)
	}
	ok = true
	return &Result{Dialect: dialect.Postgres, Realm: realm, DB: db}, nil
}

func inspectMySQL(ctx context.Context, dsn string) (*Result, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid mysql dsn: %w", err)
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return nil, fmt.Errorf("invalid mysql dsn: database name is missing (mysql://user:pass@host:port/database)")
	}
	cfg := mysqldriver.NewConfig()
	cfg.User = u.User.Username()
	cfg.Passwd, _ = u.User.Password()
	cfg.Net = "tcp"
	cfg.Addr = u.Host
	if !strings.Contains(cfg.Addr, ":") {
		cfg.Addr += ":3306"
	}
	cfg.DBName = name
	cfg.Params = map[string]string{}
	for k, v := range u.Query() {
		cfg.Params[k] = v[0]
	}
	connector, err := mysqldriver.NewConnector(cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid mysql dsn: %w", err)
	}
	db := sql.OpenDB(connector)
	ok := false
	defer func() {
		if !ok {
			_ = db.Close()
		}
	}()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connect to mysql database %q: %w", name, err)
	}
	drv, err := mysql.Open(db)
	if err != nil {
		return nil, fmt.Errorf("open mysql database %q: %w", name, err)
	}
	realm, err := drv.InspectRealm(ctx, &schema.InspectRealmOption{Schemas: []string{name}})
	if err != nil {
		return nil, fmt.Errorf("inspect mysql database %q: %w", name, err)
	}
	ok = true
	return &Result{Dialect: dialect.MySQL, Realm: realm, DB: db}, nil
}

func escapePath(p string) string {
	return (&url.URL{Path: p}).EscapedPath()
}
