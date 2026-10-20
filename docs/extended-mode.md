# Extended mode (gorm and bun)

The basic gorm/bun adapters (`schemalens-adapter-gorm`, `schemalens-adapter-bun`) read your Go
source statically and need nothing from you. **Extended mode** is a library you call from a small
program of your own: it builds the manifest from your real Go types, so the ORM itself decides every
physical name.

## What extended mode gains

- **Run-time naming is respected.** Table and column names come from the ORM's own parsing of your
  types: gorm `schema.NamingStrategy` (`TablePrefix`, `SingularTable`, `NoLowerCase`, custom
  `Namer`), `TableName()` methods that compute their result, bun's table-name inflector
  (`schema.SetTableNameInflector`), `table:` tags. Static analysis can only read literals; here
  nothing needs the `schemalens-gorm.yml` overrides.
- **Real types.** gorm's `schema.Parse` and bun's `schema.Tables` see what the ORM sees: embedded
  structs, serializers, `Scanner`/`Valuer` types, generics, types from other modules, tags
  resolved by the ORM's own parser instead of ours. Many-to-many join tables are derived by gorm.

## What it loses

- **No source positions.** Findings name tables and columns but carry no `file:line`. Use the basic
  adapter if you want findings annotated in the diff.
- **It needs a user program.** You write and maintain a `main` package that imports your models,
  and it must compile with your project (it runs your package initialisers). The basic adapter
  needs no code and never runs yours.
- **Models are listed explicitly.** `Manifest(...)` describes exactly the structs you pass; nothing
  is discovered.

## Modules

Three small modules, so nobody pulls an ORM they do not use (the core depends on neither):

| Module | Purpose |
| --- | --- |
| `github.com/mokevnin/schemalens/manifest` | ORM-agnostic manifest types, `New()` and `(*Manifest).Write(w)`; no dependencies. Also usable for adapters in other ORMs. |
| `github.com/mokevnin/schemalens/adapters/gormext` | `Manifest(namer schema.Namer, models ...any)`; depends on gorm only. |
| `github.com/mokevnin/schemalens/adapters/bunext` | `Manifest(dialect schema.Dialect, models ...any)`; depends on bun only. |

Naming: `<orm>ext` is the extended-mode library for `<orm>`; it is deliberately separate from
`adapters/<orm>`, the basic executable adapter, and imports nothing from it. Until the modules are
published, a program in this repository refers to them through the workspace (`go.work`) or a
`replace`.

## Use

```go
// cmd/schemalens-manifest/main.go in your project
package main

import (
	"log"
	"os"

	"github.com/mokevnin/schemalens/adapters/gormext"

	"yourapp/models"
	"yourapp/storage"
)

func main() {
	db := storage.Open() // or build your gorm.Config; only the naming matters
	m, err := gormext.Manifest(db.NamingStrategy, &models.User{}, &models.Order{})
	if err != nil {
		log.Fatal(err)
	}
	if err := m.Write(os.Stdout); err != nil {
		log.Fatal(err)
	}
}
```

bun is the same with `bunext.Manifest(db.Dialect(), &User{}, &Order{})`. A nil gorm namer means
gorm's defaults; for bun, set the inflector before the first call, exactly as your application does.

Run it in either of two ways:

```sh
# as an adapter: any executable on PATH named schemalens-adapter-<name>
go build -o ./bin/schemalens-adapter-myapp ./cmd/schemalens-manifest
PATH=./bin:$PATH schemalens check --dsn "$DATABASE_URL" --adapter myapp

# or save the document and pass it in
go run ./cmd/schemalens-manifest > manifest.json
schemalens check --dsn "$DATABASE_URL" --manifest manifest.json
```

The output is the normal manifest (`schemalens manifest-schema` prints its JSON Schema) and goes
into `check --manifest` unchanged; see [adapters.md](adapters.md) for the protocol.

## What is emitted

Columns (`required` for `not null` or key columns, `unique`, `max_length` for string columns
with a size), primary key, composite uniques, and foreign-key relations. gorm: belongs-to, has-one,
has-many keys on whichever table owns them, and many2many join tables (`join_table_for`). bun: only
the relations bun turns into foreign keys (belongs-to and has-one with a plain key column, created
by `CreateTable().WithForeignKeys()`), and a junction model carries `join_table_for`; bun creates
no foreign key for has-many or `m2m` on its own, so none is claimed.
