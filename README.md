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
  From source, `go install github.com/mokevnin/schemalens/adapters/ent/cmd/schemalens-adapter-ent@<version>`,
  and `go install github.com/mokevnin/schemalens/adapters/gorm@<version>` (same for bun). The gorm
  and bun installs need a release newer than v0.1.0, which still carries a local `replace`; use its
  archives.

How releases are made: [docs/releasing.md](docs/releasing.md).

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
