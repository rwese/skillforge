---
id: 2026-06-22-IFPCMUWE-PRIEJ
title: ship a self-describing SKILL.md for skillforge
status: now
created: 2026-06-22
---

# ship a self-describing SKILL.md for skillforge

## Context

Add a skill for the skillforge repo at `skill/SKILL.md` so AI agents
installing the skillforge repo can discover how to install the CLI,
search for skills, and load a skill into a temp directory. Ships
with the repo so users get agent instructions when they pull it.

## Scope

**In Scope:**

- New file `skill/SKILL.md` at the repo root (per user direction).
- Content covers: installing skillforge, `skill search`, `skill load`.
- Run `quick_validate.py` against the new skill.

**Out of Scope:**

- Extending the existing global skill at
  `~/.pi/agent/skills/skillforge/SKILL.md` (user picked "ship in
  this repo" over extending).
- Documenting `skill install`, `skill list`, `skill remove`,
  `repo *`, `sync`, `target *`, or `setup` beyond passing mentions
  needed to make search/load understandable.

## Acceptance Criteria

- [ ] `skill/SKILL.md` exists at the repo root.
- [ ] YAML frontmatter has `name` and `description` (per
      skill-creator guidance).
- [ ] Body covers: install (3 methods), search (one example),
      load (output format, hash layout, cleanup note).
- [ ] `quick_validate.py skill/` passes.
- [ ] `go build ./cmd/skillforge/` still passes (no code touched).

## First Verifiable State

- [ ] First task: write `skill/SKILL.md`, then
      `python quick_validate.py skill` returns success.

## Implementation Notes

- Naming: `name: skillforge` so it triggers on the same keyword
  as the existing global skill, matching the repo's name.
- Description must include the trigger phrases: install,
  search, load, skillforge.
- Keep SKILL.md under ~150 lines per skill-creator's
  "concise is key" principle.
- Quick reference table at the top for easy scanning.
- Reference the `skill load` temp-dir layout exactly as the
  CLI's `--help` describes it so docs and behavior stay in
  sync.

## Incremental Plan

1. Write `skill/SKILL.md`.
2. Validate with `quick_validate.py`.

## Definition of Done

- [ ] Skill validated
- [ ] No code changes (this is docs-only)
## Result

Added `skill/SKILL.md` at the repo root (137 lines) covering:

- Quick reference table.
- Three install methods (brew, go install, git+install.sh).
- First-time setup via `skillforge setup`.
- Search (`skillforge skill search <query>`) including nested-name
  matching and `--format json`.
- Load (`skillforge skill load <name>`) including the temp-dir
  layout (`/tmp/skillforge-<hash>/<skill-name>/` with `xxx-xxx`
  hash) and the cleanup note.
- Global flags and common workflows.

Validated with `quick_validate.py`: PASS (only warning is the
optional `agents/openai.yaml`, which the user did not request).
No code changes; `go build ./cmd/skillforge/` still passes.
