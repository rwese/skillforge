# Publishing a Release

This document is the single source of truth for cutting a skillforge
release. Follow it in order; deviations need an explicit reason.

## Semver

Tags follow `v{major}.{minor}.{patch}`.

- `MAJOR` (X.0.0): breaking CLI/contract changes. Requires an `[Unreleased]`
  entry in `CHANGELOG.md` with `### Changed` + `**Breaking**` notes.
- `MINOR` (0.X.0): new feature, command, flag, or non-breaking capability.
- `PATCH` (0.0.X): bug fix, performance improvement, or docs-only change.

When in doubt, default to `PATCH` (the smallest bump that fits the
change). Never bump `MAJOR` and `MINOR` in the same release.

## Pre-flight

Before cutting a tag:

1. Working tree is clean (`git status` reports nothing to commit).
2. On `main` and tracking `origin/main`:
   `git fetch origin && git status` shows `up to date with origin/main`.
3. `go build -o skillforge ./cmd/skillforge/` succeeds.
4. `go test ./...` is green.
5. `go vet ./...` and `gofmt -l ./...` are clean (no output).
6. `CHANGELOG.md` `[Unreleased]` has a complete entry for the change(s)
   shipping in this release.

## Release steps

```bash
# 1. Confirm state
git status
git log -1 --format="%H %s"   # should be the latest commit on main

# 2. Bump CHANGELOG.md
#    Edit CHANGELOG.md: rename the heading
#      ## [Unreleased]
#    to
#      ## [vX.Y.Z] - YYYY-MM-DD
#    and add a fresh empty
#      ## [Unreleased]
#    section above it. Use today's date.

# 3. Commit the version bump
git add CHANGELOG.md
git commit -m "chore: release vX.Y.Z"

# 4. Tag (annotated, signed if you have a signing key configured)
git tag -s -a vX.Y.Z -m "vX.Y.Z"
# If you don't have a signing key, drop -s:
#   git tag -a vX.Y.Z -m "vX.Y.Z"

# 5. Verify
git tag --list "vX.Y.Z" -n1
git show vX.Y.Z --no-patch

# 6. Push the commit and the tag
git push origin main
git push origin vX.Y.Z
```

After the push, the release is cut. There are no further CI or publish
steps in this repository.

## Rollback

If you cut a tag in error, before anyone consumes it:

```bash
git tag -d vX.Y.Z
git push origin :refs/tags/vX.Y.Z
```

If the tag has already been fetched downstream, do **not** rewrite it.
Cut a follow-up `PATCH` release that reverts or fixes the bad change
instead, and document it in `CHANGELOG.md`.

## Changelog conventions

- One `### Added` / `### Changed` / ### Fixed` / ### Removed` block per
  release; bullet points inside, terse, present tense.
- `### Changed` is preferred over `### Fixed` for behavior changes that
  are not strictly bug fixes.
- A `**Breaking**` prefix on a bullet under `### Changed` is required
  for any change that breaks the CLI contract, config schema, or on-disk
  layout.
- Reference the user-visible effect, not the implementation detail.
