# Writing an adapter

An adapter teaches schemalens about one ORM or model framework (ent, gorm, bun, ...). It is an
ordinary executable, in any language, that reads the project's models and prints a manifest.
schemalens itself never loads ORM code.

## Invocation

```
schemalens check --dsn <dsn> --adapter ent [--adapter-arg <arg>]...
```

`--adapter <name>` runs the executable `schemalens-adapter-<name>`, looked up on `PATH`.

- **Working directory:** the directory schemalens was started in. That is the project; the adapter
  finds its models relative to it. There is no project-path argument.
- **Arguments:** none by default. Every `--adapter-arg <value>` is passed through, in order, as one
  argument. schemalens does not interpret them; use them for adapter-specific settings such as a
  package path (`--adapter-arg ./ent/schema`).
- **Environment:** inherited unchanged. schemalens sets no variables.
- **Standard input:** empty.
- `--adapter` and `--manifest` are mutually exclusive.

## Output

On success the adapter prints exactly one manifest document (JSON) to standard output and exits 0.
Print nothing else to stdout. The document is validated like a manifest file given to
`--manifest`: the `version` must be one this schemalens supports, and the document must match the
JSON Schema printed by

```
schemalens manifest-schema
```

Use that schema to validate your output while developing. Invalid output fails the run with a
message naming the adapter and the violation.

## Exit codes and errors

- Exit **0**: stdout holds the manifest.
- Exit **non-zero** (any value): the adapter failed. schemalens stops with exit code 2 and prints
  the adapter's standard error after a line naming the executable and its exit status. Write a
  human-readable cause to stderr ("cannot load ./ent/schema: ...").
- Standard error is only shown on failure; diagnostics from a successful run are discarded.

## When the adapter is missing

If no `schemalens-adapter-<name>` is on `PATH`, schemalens exits 2 with an error naming the
executable and the `PATH` it searched.

## Minimal example

```sh
#!/bin/sh
# schemalens-adapter-example
cat <<'JSON'
{"version": 1, "tables": [{"name": "users", "columns": [{"name": "email", "required": true}]}]}
JSON
```
