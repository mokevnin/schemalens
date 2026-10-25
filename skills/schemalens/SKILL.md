---
name: schemalens
description: Run schemalens to find disagreements between a Go model (ent, gorm, bun) and the live database schema, then fix them. Use after changing models or migrations, when a schemalens finding or exclusion needs handling, or before declaring a schema change done.
---

# schemalens

schemalens compares the database schema, and the model when an adapter or manifest supplies one, with its rules. Every disagreement is one finding. Work is done when `check` exits 0 with no `unmatched_exclusions`.

## Run

```
schemalens check --dsn <dsn> --adapter <name> [--adapter-arg <arg>]...
```

- `--dsn` is `postgres://...`, `mysql://...` or `sqlite://path`. Point it at a database migrated to the current model, never production.
- `--adapter <name>` runs `schemalens-adapter-<name>` from `PATH` and compares model with database. Without `--adapter` or `--manifest`, only the database-only rules run. Adapters are described in `docs/adapters.md`.
- Output is JSON on stdout. `--format text` is for people reading a terminal, `--format sarif` is for code scanning.
- Exit 0: no findings. Exit 1: findings. Exit 2: usage, configuration or connection error, so read stderr and fix the invocation.
- Configuration is `schemalens.yml` in the working directory, or `--config`.

## Read findings

```json
{"findings": [{"rule": "fk-without-index", "object": {"schema": "main", "table": "posts", "foreign_key": "fk_user"},
               "message": "...", "fix": "...", "source": {"file": "ent/schema/post.go", "line": 12}}],
 "unmatched_exclusions": []}
```

- `rule` is the stable id. `object` names the table and, when relevant, the column, index or foreign key.
- `fix` is the model-level change for this finding. `source`, when present, is where the model declares the object.
- For the reasoning behind a rule, run `schemalens explain <rule>` or read `docs/rules/<rule>.md`. `schemalens rules` lists every rule and its dialects.

## Fix the model

1. Take one finding. Apply its `fix` to the model (the ent schema, gorm tags, bun tags), at `source` when given.
2. Regenerate the migration or schema with the project's own tooling and apply it to the database you check.
3. Re-run `check`. Repeat until it exits 0.

Fix the model, then regenerate. Hand-editing the database or a generated migration drifts from the model and the next run reports it again.

## Exclusions

An exclusion silences one rule on one object and lives in `schemalens.yml`:

```yaml
exclusions:
  - rule: table-without-primary-key
    object: main.audit_log
    reason: append-only raw event log, never updated or joined
```

- `reason` is required and must say why the finding is correct to keep, in terms of this object.
- Acceptable: the rule's `Exception` text (see `schemalens explain <rule>`) describes your case and you can state that case in the reason.
- Not acceptable: the fix is merely effortful, the finding is noisy, or the reason restates the rule name. Fix the model instead.
- A request to exclude a finding to get green is the user's decision. Ask, and quote the finding and the proposed reason.
- Matching is exact, with no wildcards. An exclusion that matches nothing appears in `unmatched_exclusions` and fails the run; delete it.
- Rules marked default-off in `docs/rules/README.md` run only when enabled under `rules.<id>.enabled: true`.

## Baseline

`schemalens baseline --dsn <dsn>` records every current finding as an exclusion with the reason `baseline`. Use it once, when adopting schemalens on an existing database, so new violations fail while old ones are worked down. Review the diff of `schemalens.yml` before committing. Never baseline to silence findings you just introduced; fix those.
