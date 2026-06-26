# skillforge Agent Instructions

Go CLI for managing agent skills from git repositories.

## Commands

```bash
go build -o skillforge ./cmd/skillforge/  # Build binary
go test ./...                              # Run tests
go test ./... -cover                       # Coverage
go mod tidy                                # Update dependencies
./skillforge sync                          # Check global target sync
./skillforge sync --fix-all                # Update repos and fix global target sync
./skillforge sync --fix-broken-symlinks    # Re-link broken skill symlinks (local+global)
```

## Structure

```text
skillforge/
├── cmd/skillforge/      # cobra CLI commands and command tests
├── internal/
│   ├── agents/          # agents.toml setup/detection helpers
│   ├── config/          # config.toml loading, saving, scope detection
│   ├── repo/            # git cache, skill discovery, install/link metadata
│   └── search/          # simple keyword search
├── pkg/grimoire/        # public grimoire metadata types
└── docs/                # PRDs, plans, reviews
```

## Code Style

- Go 1.21+.
- cobra for CLI structure.
- TOML for config.
- Use `os/exec` for git; do not add external git libraries.
- Keep search simple; do not add external search libraries.
- Prefer backwards-compatible config changes.

## Key Implementation Notes

- Skill discovery walks `<repo>/skills/` and `<repo>/.agents/skills/` recursively. A directory containing `SKILL.md` (and no `.grimoire` marker) is a skill; `Skill.Name` is its path relative to the skills root using forward slashes. A flat skill at `<repo>/skills/foo` is named `foo`; a nested skill at `<repo>/skills/architecture/event-sourced-commands` is named `architecture/event-sourced-commands`. The walker stops descending once a skill directory is found (skills never contain skills). The legacy fallback that scans `<repo>/<name>/` directly (one level deep) is preserved for caches that pre-date the `skills/` wrapper.
- All consumer code paths (`ListInstalledSkills`, `ListInstalledSkillsSymlinks`, `FindBrokenSymlinks`, `collectSkillNames`, `LinkSkill`) walk recursively with the same stop-at-skill-boundary logic; `Skill.Name` is always the slash-joined relative path. `search.matchScore` matches each path segment of `Skill.Name` independently so a query of `architecture` finds `architecture/event-sourced-commands`.
- Active install/list/remove behavior uses `config.Target` from `config.toml`.
- `target.globalPath` is the legacy global directory.
- `target.globalPaths` is a named map of additional global directories.
- If `globalPaths` is empty, treat `globalPath` as named `default`.
- `sync` is global-only and read-only by default; `--fix-sync-repos` updates repos, `--fix-outofsync-agents` links missing skills, `--fix-broken-symlinks` re-links broken skill symlinks in any target (local or global) using an absolute path into the cache, and `--fix-all` applies all three.
- `setup` uses `internal/agents` and `agents.toml`; do not assume that path drives `skill` commands.
- The on-disk git cache is a single shared directory. The effective `cache.path` is: local override > global override > default. All commands that read or write the cache (`repo add/list/remove/update`, `skill install`, `skill search`, completions, `sync`) must go through `config.EffectiveCachePath` (file-based) or `config.EffectiveCachePathFromConfigs` (in-memory) — never read `cfg.Cache.Path` directly, since `Load()` pre-populates it with the default and that hides whether the user actually set the field.

## Testing

- Run `go test ./...` before committing.
- Run `go build -o skillforge ./cmd/skillforge/` before committing.
- Manually exercise CLI behavior after command-signature, config, install, remove, or sync changes.
- Add regression tests before fixing bugs.

## Git Workflow

- Branches: `feat/<name>`, `fix/<name>`.
- Commits: Conventional Commits.
- Release tags: `v{major}.{minor}.{patch}` only, for example `v0.10.0`.
- PRs: full merge into `main` — **no squash, no rebase**. The
  feature branch's commit history is preserved on `main` so each
  commit can be inspected, reverted individually, or cited in
  follow-ups.
- After a PR is fully approved, merge the branch into `main`
  (e.g. `gh pr merge --merge` on the remote, or
  `git switch main && git merge --no-ff <branch>` locally).
  Then verify on `main`:
  - All feature-branch commits landed (`git log --oneline
    origin/<branch>..main` shows nothing, `git log main..origin/<branch>`
    shows nothing).
  - No unresolved conflicts (working tree clean, no merge
    markers anywhere).
  - `main` is up to date with the merged branch
    (`git log --oneline -1` matches the expected tip).
  After verification, end on `main` with a clean working tree.
  Do NOT leave the agent on the feature branch.

## Publishing / Release

The full release flow is described in `docs/publish.md` (read it before
publishing). Short version:

1. `git status` clean, on `main`, up to date with `origin/main`.
2. `CHANGELOG.md` `[Unreleased]` section has the entry for this release.
3. Bump the version header from `[Unreleased]` to `[vX.Y.Z] - YYYY-MM-DD`
   (semver: `MAJOR` for breaking, `MINOR` for new feature, `PATCH` for
   bug fix).
4. Commit the version bump (`chore: release vX.Y.Z`).
5. Tag: `git tag -s -a vX.Y.Z -m "vX.Y.Z"`.
6. Push: `git push origin main && git push origin vX.Y.Z`.
7. Done. No further CI/release steps in this repo.

## Boundaries

**ALWAYS**

- Use absolute paths in agent responses.
- Respect pre-commit hooks and fix their root causes.
- Build and test after config-related changes.
- Run `go build` before committing.
- Get a positive review from the `reviewer` subagent on the staged
  diff before committing. Warnings raised by the reviewer MUST be
  addressed or explicitly justified in the commit body before the
  commit is made.

**USUALLY / ASK FIRST**

- Add new dependencies.
- Modify config format.
- Change command signatures.
- Delete files.

**NEVER**

- Commit secrets or API keys.
- Modify shared user config without approval.
- Bypass or disable quality gates.
