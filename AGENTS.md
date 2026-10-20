# AGENTS.md

Guidance for coding agents working in this repository.

## What this is

`schemalens` is an agent-oriented database consistency checker for Go: it compares a model
(ent, gorm, bun, ...) with a live database schema and reports every disagreement as a structured
finding. The core knows databases (through Atlas's inspection library), adapters know ORMs and
print a versioned JSON manifest. The full design is the spec in issue #1.

This is a standalone project. It imports nothing from sphericon and uses none of its vocabulary.

## Where things are decided

- The spec and its tickets live in GitHub Issues (`mokevnin/schemalens`), via the `gh` CLI. See
  `docs/agents/issue-tracker.md`.
- Triage labels: `ready-for-agent` marks a ticket an agent can pick up.

## Conventions

- Commit messages follow Conventional Commits (release-please reads them). History stays linear:
  rebase onto `origin/main`, never merge.
- Everything in the repo is in English.
- The single test seam is `schemalens check`: input is DDL (and a manifest), output is findings and
  the exit code. Tests assert that behavior, never which internal function ran.

## Agent skills

Engineering skills are installed from `mattpocock/skills` into `.claude/skills/` and pinned in
`skills-lock.json` (`npx skills@latest update -y` refreshes them).

- **Issue tracker:** GitHub Issues (`mokevnin/schemalens`) via the `gh` CLI. See `docs/agents/issue-tracker.md`.
- **Triage labels:** the default five-label vocabulary. See `docs/agents/triage-labels.md`.
- **Domain docs:** single-context. A `GLOSSARY.md` at the repo root and `docs/adr/`, both created when the
  first term or decision is resolved. See `docs/agents/domain.md`.
