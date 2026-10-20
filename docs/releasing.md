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
`schemalens-adapter-bun`. Libraries (`adapters/staticmodel`, extended-mode packages) are not
released as binaries.

The core module (`go.mod` at the root) requires no adapter module; a test in `internal/release`
guards that.

## Try it locally

```sh
mise run release-snapshot   # goreleaser release --snapshot --clean; archives land in ./dist
```

Nothing is published. `goreleaser check` validates the config alone.

## Decisions

- **Nested module tags.** Go resolves a nested module from tags prefixed with its directory, so the
  repo also gets `adapters/<name>/vX.Y.Z` for ent, gorm, bun and staticmodel at the release commit.
  `release.yml` pushes them (skipping ones that exist). release-please does not manage them: it
  produces the single repo version only.
- **Workspace builds.** gorm and bun require `adapters/staticmodel v0.0.0` through
  `replace ... => ../staticmodel`. Each adapter builds from its own directory with `GOWORK=off`
  (the replace resolves), and goreleaser builds from the repo root with `go.work` in effect, so it
  never depends on that. Do not set `GOWORK=off` in `.goreleaser.yaml`.
- **`go install` of gorm and bun** does not work from the proxy while the `replace` is there (Go
  rejects it). Releases ship binaries instead. Dropping the `replace` and requiring the tagged
  staticmodel version is the follow-up if that matters; the tags above make it possible.
- **Notes.** The GitHub release is created by release-please and then overwritten by goreleaser
  (`release.mode: replace`), so the notes come from commit messages, as the goreleaser config says.

## Not verified outside CI

The workflows (`release-please.yml`, `release.yml`) cannot run locally. The goreleaser snapshot, the
per-module builds and the config tests do run locally.
