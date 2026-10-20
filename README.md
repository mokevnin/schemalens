# schemalens

An agent-oriented database consistency checker for Go. It compares a model (ent, gorm, bun, ...)
with a live database schema and reports every disagreement as a structured finding.

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
