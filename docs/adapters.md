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

## Built-in adapters and shared code

Adapters live in `adapters/<name>/`, each a Go module of its own (module path
`github.com/mokevnin/schemalens/adapters/<name>`, binary `schemalens-adapter-<name>`), so the core
never depends on an ORM. All modules are listed in the `go.work` at the repo root. `mise run lint`
and `mise run test` visit every `go.mod` in the repo (testdata excluded).

Static-analysis code shared by Go adapters is the module `adapters/staticmodel`
(`github.com/mokevnin/schemalens/adapters/staticmodel`). It is deliberately not under `internal/`
(that visibility does not cross modules). It loads packages with `golang.org/x/tools/go/packages`
and offers: `Load(dir, patterns...)`, `Program.Structs()` (named structs with fields, struct tags,
types, project-relative positions), `Struct.Flatten(expand)` (embedded structs expanded; fields
declared outside the project, like `gorm.Model`, take the position of the embedding field),
`Struct.StringMethod` (a method that returns a string literal, e.g. `TableName`), `NamedType`,
`HasMethod`. It knows no ORM; each adapter decides what a model is and what its tags mean. An
adapter module requires it with a `replace ... => ../staticmodel` until the modules are published.

### gorm (`adapters/gorm`)

```
schemalens check --dsn ... --adapter gorm [--adapter-arg ./models]... [--adapter-arg --config=file.yml]
```

Arguments are Go package patterns (default `./...`, or `packages` from the config). The project is
not modified. A struct is a model when it embeds `gorm.Model`, has a `TableName` method or a config
entry, or has `gorm` tags and is not merely embedded in another struct. Understood: embedded
structs (anonymous or `embedded` with `embeddedPrefix`), `gorm.Model`, `column`, `not null`,
`primaryKey` (default key: column `id`), `unique`, `uniqueIndex` (named shared indexes become composite
uniques), `size` (string fields), `-`/`-:migration`/`-:all`. Relations are not columns: belongs-to,
has-one and has-many with `foreignKey` or gorm's default key names become manifest `relations`;
`many2many` is not covered yet. Table and column names come from gorm's own
`schema.NamingStrategy`.

What static analysis cannot see goes in `schemalens-gorm.yml` (project directory, or `--config`):

```yaml
naming:                    # schema.NamingStrategy
  table_prefix: "app_"
  singular_table: false
  no_lower_case: false
tables:                    # struct name (or importpath.Name) -> physical table name, verbatim
  Company: legacy_companies
exclude: [Scratch]         # look like models, are not
```

A `TableName` method returning a string literal is read as is; one computing its result makes the
adapter fail, naming the struct and file:line, until `tables` supplies the name.

### bun (`adapters/bun`)

```
schemalens check --dsn ... --adapter bun [--adapter-arg ./models]... [--adapter-arg --config=file.yml]
```

Same shape as the gorm adapter: arguments are Go package patterns, the project is not modified, and
the command, config loading and manifest types come from `adapters/staticmodel`. A struct is a
model when it embeds `bun.BaseModel`, has a config entry, or has `bun` tags and is not merely
embedded in another struct. Understood: `bun.BaseModel` tags (`table:name`, optionally
`schema.name`), anonymous embedded structs and `embed:prefix_`, the column name (`bun:"name"` or
`column:`), `pk` (primary key, implies not null), `notnull`, `autoincrement`, `unique` (plain, or
`unique:group` shared across fields into a composite unique), `type:varchar(n)` / `char(n)` (as
`max_length`), `-`, `scanonly` (skipped). Relations are not columns: `rel:belongs-to`, `has-one`
and `has-many`, with `join:base=joined` or bun's default key names, become manifest `relations`;
`m2m` and `polymorphic` are not covered. A relation naming a column the struct lacks fails the
run (bun would panic at start-up), naming file:line of the field. Table names follow bun: the
`BaseModel` tag, else the struct name underscored and pluralized (`UserProfile` -> `user_profiles`).

What static analysis cannot see goes in `schemalens-bun.yml` (project directory, or `--config`):

```yaml
naming:
  singular_table: false    # true if the project sets an identity schema.SetTableNameInflector
tables:                    # struct name (or importpath.Name) -> physical table name, verbatim
  Label: tags
exclude: [Scratch]         # look like models, are not
```

The adapter's tests check its manifest against bun's own runtime schema for the sample project
(table names, columns, not-null, primary keys).

### Extended mode

For run-time naming and real types, `adapters/gormext` and `adapters/bunext` build the manifest from your
compiled models inside a program you write, on top of the ORM-agnostic `manifest` module. See
[extended-mode.md](extended-mode.md).
