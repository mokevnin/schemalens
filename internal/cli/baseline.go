package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/mokevnin/schemalens/internal/config"
	"github.com/mokevnin/schemalens/internal/inspect"
	"github.com/mokevnin/schemalens/internal/rules"
)

// runBaseline records the current findings as exclusions with the reason
// "baseline" in the configuration file, creating it if needed. Findings that
// are already excluded are left alone, so it is safe to run repeatedly.
func runBaseline(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("baseline", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dsn := flags.String("dsn", "", "database to baseline, for example sqlite://path/to/db.sqlite")
	configPath := flags.String("config", config.DefaultFile, "configuration file to update (created if missing)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitUsage
	}
	if *dsn == "" {
		_, _ = fmt.Fprintln(stderr, "schemalens baseline: --dsn is required")
		return ExitUsage
	}
	data, err := os.ReadFile(*configPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		_, _ = fmt.Fprintf(stderr, "schemalens baseline: %v\n", err)
		return ExitUsage
	}
	cfg, err := config.Parse(data)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens baseline: config %s: %v\n", *configPath, err)
		return ExitUsage
	}
	res, err := inspect.Inspect(context.Background(), *dsn)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens baseline: %v\n", err)
		return ExitUsage
	}
	findings := rules.RunEnabled(rules.Input{Dialect: res.Dialect, Realm: res.Realm, RuleOptions: cfg.Options}, cfg.Enabled)
	out, added, err := config.AddBaseline(data, findings)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "schemalens baseline: %v\n", err)
		return ExitUsage
	}
	if added > 0 {
		if err := os.WriteFile(*configPath, out, 0o644); err != nil { //nolint:gosec // a config file meant to be committed
			_, _ = fmt.Fprintf(stderr, "schemalens baseline: %v\n", err)
			return ExitUsage
		}
	}
	_, _ = fmt.Fprintf(stdout, "baseline: added %d exclusion(s) to %s\n", added, *configPath)
	return ExitOK
}
