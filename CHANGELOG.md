# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- `repo add` now checks for duplicates per config scope instead of per
  on-disk cache: adding a repository to the local scope that is already
  registered in the global scope is valid (and vice versa). The existing
  shared cache entry is reused instead of cloning again. A same-name
  cache entry cloned from a different URL is never reused; the command
  fails with a hint to remove the conflicting entry first.
- `skill install` always sources skills from the local scope's
  repositories first (falling back to the global scope); the `--scope`
  flag now only selects the install targets, not the source.
- `repo.Pull` now fast-forwards the local cache branch to
  `origin/<branch>` via `git reset --hard` instead of plain
  `git pull`. The cache is a shallow clone and `Fetch` uses
  `--depth 1`, so the local branch and the freshly-fetched tip
  could be git-unaware of any common ancestor and `git pull`
  would fail with "Need to specify how to reconcile divergent
  branches", leaving the cache permanently stale. `skillforge
  sync --fix-sync-repos` and `skillforge repo update` now reliably
  advance a stale cache to the latest remote tip.

### Added
- `skill export <name> <destination>` copies a cached skill into a
  NEW directory the caller specifies. The destination must not
  already exist (file, directory, symlink, or broken symlink); if
  it does, the command refuses to overwrite or merge. Missing
  parent directories along the destination path are created. The
  skill's files land directly under `<destination>`; no
  skill-name wrapper subdirectory is created. Nested skill names
  like `architecture/event-sourced-commands` resolve to the same
  on-disk skill and export with their source files directly under
  the destination. Useful for materializing a skill into a chosen
  path for sharing, hand-off, or inspection. The export never
  carries a `.grimoire` marker (install metadata is local-only).
- `skill load <name>` copies a cached skill into a fresh temp
  directory and prints its `SKILL.md`. The temp dir is laid out as
  `/tmp/skillforge-<hash>/<skill-name>/` where `<hash>` is the first
  three bytes of the SHA256 of the current working directory,
  formatted as `xxx-xxx` for readability. Nested skill names keep
  their category path on disk, so
  `skillforge skill load architecture/event-sourced-commands` lands
  at `/tmp/skillforge-<hash>/architecture/event-sourced-commands/`.
  The directory is left in place; the user is responsible for
  removing it. Useful for inspecting or handing off a skill without
  linking it into an agent's skills directory.

## [v0.11.0] - 2026-06-18

### Added
- Nested-skill discovery: `repo.DiscoverSkills` now walks
  `<repo>/skills` and `<repo>/.agents/skills` recursively. Any
  directory containing `SKILL.md` (and no `.grimoire` marker) is a
  skill, and `Skill.Name` is its path relative to the skills root
  using forward slashes. A skill at
  `<repo>/skills/architecture/event-sourced-commands` is therefore
  named `architecture/event-sourced-commands`, while a flat skill at
  `<repo>/skills/docker` keeps the bare name `docker`. The legacy
  fallback (skills placed directly under `<repo>`) is preserved for
  backwards compatibility.
- Recursive `repo.ListInstalledSkills`,
  `repo.ListInstalledSkillsSymlinks`, and `repo.FindBrokenSymlinks`
  so installed-skill listings and broken-symlink scans discover
  nested installs (`<target>/<category>/<name>`). The `Name` field
  is the slash-joined relative path so catalog lookups match.
- `cmd.collectSkillNames` walks recursively for the same reason;
  nested installs are reported under their slash-joined relative
  path in `sync --fix*` flows.
- `sync --fix-broken-symlinks` to re-link broken skill symlinks in local
  and global targets. Broken symlinks whose names match a skill in the
  cache are removed and re-linked with an absolute path; broken symlinks
  with no matching skill are reported and left in place. `sync --fix-all`
  now also enables this step.
- `repo.IsBrokenSymlink`, `repo.FindBrokenSymlinks`, and
  `repo.RellinkSkill` helpers in `internal/repo` for the fix flow.
- `--version` flag (cobra-managed) reports the tagged release.

### Changed
- `search.matchScore` now matches each path segment of `Skill.Name`
  independently. A nested name like `architecture/event-sourced-commands`
  is matched by either its category or its leaf segment, so a query
  of `architecture` finds the skill (previously the legacy matcher
  only saw the full slash-joined Name).

### Fixed
- `repo.IsBrokenSymlink` previously called `os.Readlink`, which always
  succeeds for a valid symlink and therefore reported *every* symlink
  as not broken. It now calls `os.Stat` (which follows the link) so a
  dangling target is correctly identified.
- `sync --fix-broken-symlinks` repairs relative symlinks created by
  the previous `LinkSkill` (e.g. when a project like
  `git.void.cold.at/nope-at/Devops` is moved to a new parent
  directory and the relative paths no longer resolve).

## [v0.10.1] - 2026-06-04

### Fixed
- `skill install -s local` (and other scope-aware commands) now resolve the
  on-disk git cache via a single effective `cache.path` instead of
  disagreeing between the global and local configs. Previously, the global
  config's custom `cache.path` was used to look for repos even when the
  local config had no `cache.path` override, so a repo that
  `repo add -s local` had cloned to the default location was reported as
  "not found in any cached repository".

## [v0.10.0] - 2026-06-03

### Added
- `sync --diff` to show incoming remote commits and changed files.
- `sync --fix-sync-repos`, `sync --fix-outofsync-agents`, and `sync --fix-all` for explicit fix controls.

### Changed
- `sync` now fetches cached repos in read-only mode and reports remote changes.
- Local skill installs now use globally configured targets with `localPath` when running inside a git repo.
- Release tags use `v{major}.{minor}.{patch}` format.

### Removed
- Removed broad `sync --fix` and `sync --skip-agent-sync` flags.

### Changed
- **Breaking**: Skills are now symlinked instead of copied
  - No more `.grimoire` metadata files
  - Skills always reference latest version from cached repos
  - Removed `skill update` command (auto-updates on `sync`)
- docs: slim down v2 scope (YAGNI - removed TUI, suggest, plugin system)

## [v1.1] - 2026-04-28

### Added
- `--dry-run` flag for previewing operations
- `-y/--yes` flag for skipping confirmations
- `-v/--verbose` flag for debug output
- Confirmation prompts for destructive actions (remove commands)
- Progress reporting for file copy operations (with `--verbose`)
- Comprehensive test suite (config, search, grimoire, repo)

### Fixed
- Cache clone branch logic for non-main branches
- Spinner goroutine leak (added thread-safe stopping)
- Unused commit variable in repo add output
- Target directory creation before skill installation

### Changed
- Improved error messages with context

## [v1.0.0] - 2024-04-28

### Added
- Repository management (add, list, remove, update)
- Skill installation from cached repositories
- Skill search across repositories
- Target management for multiple agents
- Config scope detection (global/local/auto)
- JSON output format for all list commands
- Shell completion support
- Grimoire metadata for version tracking

### Features
- Cache git repositories with shallow clones
- Discover skills in `skills/` and `.agents/skills/` directories
- Auto-detect local config in git repositories
- Progress spinner for long operations
- Skill update checking and installation
