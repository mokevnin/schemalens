# Releasing

One version for the whole repository. Conventional Commits on `main` drive everything.

## Flow

1. Merge Conventional Commits to `main`.
2. `release-please.yml` keeps a release PR open (version bump plus `CHANGELOG.md`, from
   `release-please-config.json` and `.release-please-manifest.json`). `feat` bumps minor, `fix`
   bumps patch, `!` or `BREAKING CHANGE` bumps major (minor while the version is 0.x).
3. Merging the release PR creates the tag `vX.Y.Z` and a GitHub release.
4. In the same run, `release.yml` is called with that tag. It runs `goreleaser release`
   (`.goreleaser.yaml`) and pushes the nested module tags (below). Goreleaser attaches the archives
   and replaces the release notes with its own changelog, built from commit messages and grouped by
   type (breaking changes, features, fixes, performance, other; `docs`, `test`, `chore`, `ci` are
   left out).

`release.yml` is called directly instead of listening for the tag because tags pushed with
`GITHUB_TOKEN` do not trigger workflows. It can also be run by hand (`workflow_dispatch`, with an
existing tag) to retry a failed release.

## What is released

Archives per platform (linux, darwin, windows; amd64, arm64), each its own build and archive so the
core installs alone: `schemalens`, `schemalens-adapter-ent`, `schemalens-adapter-gorm`,
`schemalens-adapter-bun`. Each adapter binary is built from `adapters/<name>/cmd/schemalens-adapter-<name>`, so
`go install <module>/cmd/schemalens-adapter-<name>@<version>` produces the same name. Libraries
(`adapters/staticmodel`, extended-mode packages, the adapter root packages) are not released as
binaries.

The core module (`go.mod` at the root) requires no adapter module; a test in `internal/release`
guards that.

## Try it locally

```sh
mise run release-snapshot   # goreleaser release --snapshot --clean; archives land in ./dist
```

Nothing is published. `goreleaser check` validates the config alone.

## Decisions

- **Nested module tags.** Go resolves a nested module from tags prefixed with its directory, so the
  repo also gets `<dir>/vX.Y.Z` for every nested module (`adapters/ent`, `adapters/gorm`,
  `adapters/bun`, `adapters/staticmodel`, `adapters/gormext`, `adapters/bunext`, `manifest`) at the
  release commit. Add new nested modules to the list in `release.yml`.
  `release.yml` pushes them (skipping ones that exist). release-please does not manage them: it
  produces the single repo version only.
- **Module dependencies.** gorm and bun require a tagged `adapters/staticmodel`, and gormext and
  bunext a tagged `manifest`; there are no `replace` directives, so every module builds from its
  own directory with `GOWORK=off` and `go install <module>@<version>` works. Inside the repository
  `go.work` resolves those requirements to the local copies, and goreleaser builds from the repo
  root with `go.work` in effect. Do not set `GOWORK=off` in `.goreleaser.yaml`.
- **Bumping a library.** After a change to `staticmodel` or `manifest` that an adapter needs,
  release it first, then raise the `require` in the dependent modules to that tag
  (`GOWORK=off go mod tidy` in each) in the next commit. v0.1.0 of gorm and bun still carry the old
  `replace`, so `go install` of those two works from the next release on.
- **Notes.** The GitHub release is created by release-please and then overwritten by goreleaser
  (`release.mode: replace`), so the notes come from commit messages, as the goreleaser config says.

## Not verified outside CI

The workflows (`release-please.yml`, `release.yml`) cannot run locally. The goreleaser snapshot, the
per-module builds and the config tests do run locally.
