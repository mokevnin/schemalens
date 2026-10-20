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
  Adapters are not built by `go install` yet (gorm and bun use a local `replace`); use the release
  archives.

How releases are made: [docs/releasing.md](docs/releasing.md).

## For agents

- **Skill:** [`skills/schemalens/SKILL.md`](skills/schemalens/SKILL.md) is a user-installable agent
  skill (copy or symlink `skills/schemalens` into your agent's skills directory, for example
  `.claude/skills/`). It covers running `check`, reading findings, fixing the model, exclusions and
  baseline.
- **Instructions snippet:** [`docs/agent-instructions.md`](docs/agent-instructions.md) is a block to
  paste into `AGENTS.md` or `CLAUDE.md`.

## Documentation

- [Rules](docs/rules/README.md): one page per rule, generated from `schemalens explain`. After
  changing a rule's explanation or adding a rule, run `mise run docs`; the test suite fails when a
  page is missing or stale.
- [Writing an adapter](docs/adapters.md)
