// Package inspect reads a live database into Atlas's schema model.
package inspect

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"
	_ "modernc.org/sqlite" // registers the "sqlite" driver

	"github.com/mokevnin/schemalens/internal/dialect"
)

// Result is an inspected database.
type Result struct {
	Dialect dialect.Dialect
	Realm   *schema.Realm
}

// Inspect connects to the database named by dsn and inspects it. The DSN
// scheme selects the dialect: sqlite://<path>. The database is opened
// read-only and never created.
func Inspect(ctx context.Context, dsn string) (*Result, error) {
	scheme, rest, ok := strings.Cut(dsn, "://")
	if !ok {
		return nil, fmt.Errorf("invalid dsn %q: expected scheme://location", dsn)
	}
	switch scheme {
	case "sqlite":
		return inspectSQLite(ctx, rest)
	default:
		return nil, fmt.Errorf("unsupported dsn scheme %q (supported: sqlite)", scheme)
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
	defer func() { _ = db.Close() }()
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
	return &Result{Dialect: dialect.SQLite, Realm: realm}, nil
}

func escapePath(p string) string {
	return (&url.URL{Path: p}).EscapedPath()
}
