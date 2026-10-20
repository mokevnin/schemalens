# schemalens

An agent-oriented database consistency checker for Go. It compares a model (ent, gorm, bun, ...)
with a live database schema and reports every disagreement as a structured finding.

## Install

Every release publishes one archive per platform (linux, darwin, windows; amd64, arm64) for the core
and for each built-in adapter, all with the same version. Download them from the
[releases page](https://github.com/mokevnin/schemalens/releases).

- **Core only:** `schemalens_<version>_<os>_<arch>`. It needs no adapter: use it with a manifest
  file (`--manifest`) or with any adapter you bring yourself. From source: `go install
  github.com/mokevnin/schemalens/cmd/schemalens@latest`.
- **Adapters:** `schemalens-adapter-ent`, `schemalens-adapter-gorm`, `schemalens-adapter-bun`, each
  its own archive. Put the binary on `PATH` next to `schemalens` and select it with `--adapter`.
  From source: `go install github.com/mokevnin/schemalens/adapters/<name>/cmd/schemalens-adapter-<name>@<version>`
  (`<name>` is `ent`, `gorm` or `bun`) installs the adapter under its final name. The `cmd` path form
  for gorm and bun needs a release newer than v0.1.2; `go install` of the older gorm and bun module
  paths (v0.1.2 and before) installs binaries named `gorm` and `bun`, so use the release archives for
  those versions.

How releases are made: [docs/releasing.md](docs/releasing.md).

## Use as a library

The core is a Go package too: `go get github.com/mokevnin/schemalens`. The CLI is a thin layer over
it, so a program gets the same findings, JSON and SARIF as `schemalens check`, and the root module
depends on no ORM.

```go
rep, err := schemalens.Check(ctx, schemalens.Options{
	DSN:      "postgres://user:pass@localhost/app",
	Manifest: manifestJSON, // optional: what an adapter prints
	Config:   cfg,          // optional: schemalens.LoadConfig("schemalens.yml")
})
if err != nil {
	return err // unusable DSN, manifest or configuration
}
for _, f := range rep.Findings {
	fmt.Println(f.Rule, f.Message, f.Fix)
}
if !rep.Passed() { // findings or stale exclusions, like exit code 1
	return rep.WriteJSON(os.Stdout)
}
```

Also available: `ParseManifest`, `ParseConfig`, `Rules`, `Explain`, `Baseline` and
`Report.WriteSARIF`. See the [package documentation](https://pkg.go.dev/github.com/mokevnin/schemalens).
Rule implementations and database inspection stay internal.

## For agents

- **Skill:** [`skills/schemalens/SKILL.md`](skills/schemalens/SKILL.md) is a user-installable agent
  skill (copy or symlink `skills/schemalens` into your agent's skills directory, for example
  `.claude/skills/`). It covers running `check`, reading findings, fixing the model, exclusions and
  baseline.
- **Instructions snippet:** [`docs/agent-instructions.md`](docs/agent-instructions.md) is a block to
  paste into `AGENTS.md` or `CLAUDE.md`.

## Exit codes and stale exclusions

`check` exits 0 when it finds nothing, 1 when it reports findings or when a configured exclusion
matches nothing, and 2 on usage or configuration errors. A stale exclusion fails the run on purpose:
it means a violation was fixed or an object renamed, and left in place it would silently hide the
next violation on that object. It is listed in the JSON `unmatched_exclusions` field, on stderr, and
in `--format sarif` output as `runs[0].invocations[0]` (`executionSuccessful: false`, one
`toolConfigurationNotifications` entry per exclusion).

## Documentation

- [Rules](docs/rules/README.md): one page per rule, generated from `schemalens explain`. After
  changing a rule's explanation or adding a rule, run `mise run docs`; the test suite fails when a
  page is missing or stale.
- [Writing an adapter](docs/adapters.md)

## Inspired by

[database_consistency](https://github.com/djezzzl/database_consistency), the Ruby tool that checks
a database schema against ActiveRecord models. schemalens brings the same idea to Go, with
structured findings built for coding agents.
