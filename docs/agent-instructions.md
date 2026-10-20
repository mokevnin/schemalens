# Agent instructions snippet

Paste the block below into your project's `AGENTS.md` or `CLAUDE.md`, replacing the placeholders. For
the full workflow, install the skill in [`skills/schemalens`](../skills/schemalens/SKILL.md).

````markdown
## Database consistency (schemalens)

After changing a model or a migration, run `schemalens check --dsn <dsn> --adapter <name>` against a
migrated database. The change is done when it exits 0 with no `unmatched_exclusions`.

- Each finding carries a `fix`. Apply it to the model, regenerate the schema, re-run `check`.
- `schemalens explain <rule>` gives the reasoning; `schemalens rules` lists the rules.
- An exclusion in `schemalens.yml` needs a `reason` and is only for the case the rule's `Exception`
  text describes. Ask the user before adding one.
````

Rule pages: [`docs/rules`](rules/README.md). Writing an adapter: [`docs/adapters.md`](adapters.md).
