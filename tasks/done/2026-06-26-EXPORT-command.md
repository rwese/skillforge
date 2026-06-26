---
id: 2026-06-26-EXPORT
title: skill export command
status: now
created: 2026-06-26
---

# skill export command

## Context

Add `skillforge skill export <name> <destination>` that copies a
cached skill into a NEW directory the caller specifies. Lets users
materialize a skill into a chosen path for sharing, hand-off, or
inspection, while making it impossible to clobber existing content.

## Scope

**In Scope:**

- New subcommand `skill export <name> <destination>` under `skill`.
- Resolve the skill via the same local-first/global-fallback resolver
  used by `skill install` and `skill load`.
- Refuse if `<destination>` already exists (any kind of entry: file,
  directory, symlink, broken symlink). Do not delete, overwrite, or
  merge.
- Create `<destination>` (and any missing parents) and recursively copy
  the skill contents into it. The skill's files land directly under
  `<destination>` (no skill-name wrapper subdirectory).
- Print a success line with the resolved destination path.
- Surface a clear error and a hint when the skill is not cached.

**Out of Scope:**

- Symlink-based export.
- Exporting installed skills by name (only cached skills are exported;
  exporting an installed link by name resolves to its on-disk target
  via the existing skill resolver, so a symlink chain to the cache is
  effectively what the cache resolver returns anyway).
- Auto-cleanup of the destination.
- Multi-skill export.
- Format flag / structured output.
- Modifying the cache.

## Acceptance Criteria

- [ ] `skillforge skill export <name> <destination>` succeeds when
      `<name>` is in a cached repo and `<destination>` does not yet
      exist. The skill files (including nested subdirs) land directly
      under `<destination>`.
- [ ] The command FAILS (non-zero exit, clear error) when
      `<destination>` already exists as any filesystem entry
      (directory, file, symlink, broken symlink).
- [ ] Missing intermediate parents of `<destination>` are created.
- [ ] The copy is recursive: nested subdirectories and their files
      are preserved.
- [ ] The exported tree does not contain a `.grimoire` marker
      (we copy the skill's source files, not the install metadata).
- [ ] Relative and `~`-prefixed destinations are supported.
- [ ] Errors are clear when the skill is not cached.
- [ ] `--help` describes the command.
- [ ] Test coverage for: happy path flat skill, happy path nested
      skill, recursive copy, destination-already-exists failure
      (directory), destination-already-exists failure (regular file),
      destination-already-exists failure (symlink), missing-skill
      error path.
- [ ] `go build ./cmd/skillforge/` and `go test ./...` pass.

## First Verifiable State

**Order first, not time.** Ensure verification is the first task in sequence.

- [ ] First task leads to testable output
- [ ] How to verify: `go test ./cmd/skillforge/... -run TestSkillExport`

## Implementation Notes

- Tech decisions:
  - New file `cmd/skillforge/cmd/skill_export.go` for the command.
  - Reuse the local-first/global-fallback skill lookup pattern from
    `runSkillInstall` so resolution semantics match exactly. The
    `resolveCachedSkill` helper in `skill_load.go` already implements
    exactly this and returns a `*grimoire.Skill` with the on-disk
    cache path populated.
  - Reuse `repo.CopyDir` (already exported by `skill_load`) for the
    recursive copy. This already skips nothing — it copies the
    full on-disk tree. We must filter `.grimoire` after the copy
    (or pre-stat and skip during walk). Since `CopyDir` is shared
    with `skill load`, we keep it as-is and remove `.grimoire`
    from the destination after the copy if present. (The cache
    directory itself should not contain `.grimoire` markers —
    those are written by `WriteGrimoire` only inside installed
    skill directories — but we filter defensively.)
  - Destination resolution: pass the user input through
    `config.ExpandPath` (handles `~`), then `filepath.Abs` so the
    error and success messages show a normalized path.
  - "MUST NOT EXIST" check: use `os.Lstat` (so a broken symlink at
    the destination is treated as "exists" and rejected, matching
    the user's intent of "do not touch anything that is already
    there").
- Key files:
  - `cmd/skillforge/cmd/skill.go` — register the new subcommand in
    `init()`.
  - `cmd/skillforge/cmd/skill_export.go` — new command implementation.
  - `cmd/skillforge/cmd/skill_export_test.go` — new tests.
  - `cmd/skillforge/cmd/root.go` — long help / examples mention `export`.
  - `CHANGELOG.md` — `[Unreleased]` entry.
  - `README.md` — skill command examples.
- Tests needed:
  - Happy path: flat skill `docker` exported to a new dir; assert
    SKILL.md lands at `<dest>/SKILL.md` and matches.
  - Recursive copy: a skill with a subdir + nested file preserves
    both under `<dest>`.
  - Destination exists as directory: command fails with a clear
    "already exists" error and the destination is not modified.
  - Destination exists as regular file: command fails with the same
    clear error.
  - Destination exists as symlink: command fails (Lstat-based check
    treats symlink as existing).
  - Missing skill: command fails with the standard
    `not found in any cached repository` error.
  - Missing parents: command creates the leaf and any missing
    intermediate dirs.

## Incremental Plan

1. **[Verification First]** — scaffold `skill_export_test.go` with
   the happy path; run it red; write the command to make it green.
2. **[Core Logic]** — implement resolution, dest-exists guard,
   `MkdirAll`, `CopyDir`, output.
3. **[Edge cases]** — pre-existing dest (dir / file / symlink),
   missing skill, missing parents, `.grimoire` filter.
4. **[Polish]** — help/usage text, root long-help mention, changelog,
   README mention.

## Definition of Done

- [ ] First verification passes
- [ ] Core functionality complete
- [ ] Logic verified
- [ ] No debug code or TODOs
- [ ] `go build ./cmd/skillforge/` passes
- [ ] `go test ./...` passes
- [ ] Changes documented in `CHANGELOG.md` under `[Unreleased]`

## Result

Implemented `skillforge skill export <name> <destination>`:

- New file: `cmd/skillforge/cmd/skill_export.go` — command + skill
  resolution + destination-exists guard (`assertExportDestinationMissing`,
  uses `Lstat` so broken symlinks at the destination are rejected)
  + `~` expansion helper.
- New file: `cmd/skillforge/cmd/skill_export_test.go` — 10 tests
  (flat skill, nested skill layout, recursive copy, refuses existing
  directory/file/symlink, missing skill, creates missing parents,
  parent-is-file obstruction, ~ expansion).
- Updated: `cmd/skillforge/cmd/skill.go` (comment in `init()` noting
  that `skillExportCmd` registers itself), `cmd/skillforge/cmd/root.go`
  (commands list mentions `export`), `CHANGELOG.md` (Unreleased entry
  for `skill export`), `README.md` (skill command examples).
- Verified: `go build ./cmd/skillforge/`, `go test ./...`,
  `go test ./cmd/skillforge/... -race -run TestSkillExport`,
  manual smoke tests (8 scenarios including nested skills and
  symlink rejection).

