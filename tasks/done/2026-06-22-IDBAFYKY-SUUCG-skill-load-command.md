---
id: 2026-06-22-IDBAFYKY-SUUCG
title: skill load command
status: now
created: 2026-06-22
---

# skill load command

## Context

Add `skillforge skill load <skill_name>` that materializes a cached
skill into a fresh temp directory and prints its `SKILL.md` to stdout.
Lets users inspect or hand off a skill without linking it into an
agent's skills directory.

## Scope

**In Scope:**

- New subcommand `skill load <skill_name>` under `skill`.
- Locate the skill using the same local-first/global-fallback resolver
  as `skill install`.
- Copy the skill directory recursively into a temp directory.
- Print the absolute temp directory path and the `SKILL.md` contents.

**Out of Scope:**

- Auto-cleanup of the temp directory.
- Symlink-based loading.
- Loading skills from arbitrary local paths (only cached skills).
- Modifying the cache or any installed skill.

## Acceptance Criteria

- [ ] `skillforge skill load <name>` succeeds when `<name>` is in a
      cached repo, prints the resolved temp dir, and the SKILL.md
      content of the loaded skill.
- [ ] Temp dir layout is
      `/tmp/skillforge-XXXXXX/<slash-joined-skill-name>/` where the
      6 hex chars are derived from the SHA256 of the cwd, formatted
      as `xxx-xxx` for readability.
- [ ] The copy preserves the full skill tree (recursive).
- [ ] Errors are clear when the skill is not cached.
- [ ] `--help` describes the command.
- [ ] Test coverage for: nested skill (`a/b`), missing skill, recursive
      copy.
- [ ] `go build` and `go test ./...` pass.

## First Verifiable State

**Order first, not time.** Ensure verification is the first task in sequence.

- [ ] First task leads to testable output
- [ ] How to verify: `go test ./cmd/skillforge/... -run TestSkillLoad`

## Implementation Notes

- Tech decisions:
  - New file `cmd/skillforge/cmd/skill_load.go` for the command +
    a small `repo.CopyDir` helper colocated next to other repo
    helpers in `internal/repo/`.
  - Reuse the local-first/global-fallback skill lookup pattern from
    `runSkillInstall` so resolution semantics match exactly.
  - Hash: `crypto/sha256` of `os.Getwd()`, first 3 bytes hex-encoded
    formatted `xxx-xxx`.
  - Skill subdir under temp uses `filepath.FromSlash(skill.Name)` so
    nested skills keep their category path on disk.
  - Print SKILL.md via `os.ReadFile(filepath.Join(skill.Path, "SKILL.md"))`.
- Key files:
  - `cmd/skillforge/cmd/skill.go` — register the new subcommand in
    `init()`.
  - `cmd/skillforge/cmd/skill_load.go` — new command implementation.
  - `cmd/skillforge/cmd/skill_load_test.go` — new tests.
  - `internal/repo/copy.go` — `CopyDir` helper (small).
  - `CHANGELOG.md` — `[Unreleased]` entry.
- Tests needed:
  - Successful load of a flat skill (assert temp dir layout and
    SKILL.md content).
  - Successful load of a nested skill `a/b` (assert dir layout is
    `<tmp>/a/b/`).
  - Missing skill error path.
  - Recursive copy: a skill with a subdir + file preserves both.

## Incremental Plan

1. **[Verification First]** — scaffold `skill_load_test.go` with the
   flat-skill happy path; run it red; write the command to make it
   green.
2. **[Core Logic]** — implement resolution, temp dir creation,
   `CopyDir`, and output.
3. **[Polish]** — nested-skill layout, missing-skill error,
   help/usage text, changelog entry.

## Definition of Done

- [ ] First verification passes
- [ ] Core functionality complete
- [ ] Logic verified
- [ ] No debug code or TODOs
- [ ] `go build ./cmd/skillforge/` passes
- [ ] `go test ./...` passes
- [ ] Changes documented in `CHANGELOG.md` under `[Unreleased]`
## Result

Implemented `skillforge skill load <name>`:

- New file: `cmd/skillforge/cmd/skill_load.go` — command + skill
  resolution + cwd-hash helper.
- New file: `cmd/skillforge/cmd/skill_load_test.go` — 4 tests
  (flat skill, nested layout, recursive copy, missing skill).
- Refactor: `internal/repo/grimoire.go` exports `CopyDir` (was
  unexported `copyDir`) so the cmd package can use it; the one
  in-package test caller was updated.
- Updated: `cmd/skillforge/cmd/root.go` (commands list mentions
  load), `CHANGELOG.md` (Unreleased entry).
- Verified: `go build ./cmd/skillforge/`, `go test ./...`,
  `go test ./cmd/skillforge/... -race -run TestSkillLoad`, manual
  smoke test.
