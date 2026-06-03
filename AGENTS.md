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

- Skill discovery checks `skills/` and `.agents/skills/` in cached repos.
- Active install/list/remove behavior uses `config.Target` from `config.toml`.
- `target.globalPath` is the legacy global directory.
- `target.globalPaths` is a named map of additional global directories.
- If `globalPaths` is empty, treat `globalPath` as named `default`.
- `sync` is global-only and read-only by default; `--fix-sync-repos` updates repos, `--fix-outofsync-agents` links missing skills, and `--fix-all` applies both.
- `setup` uses `internal/agents` and `agents.toml`; do not assume that path drives `skill` commands.

## Testing

- Run `go test ./...` before committing.
- Run `go build -o skillforge ./cmd/skillforge/` before committing.
- Manually exercise CLI behavior after command-signature, config, install, remove, or sync changes.
- Add regression tests before fixing bugs.

## Git Workflow

- Branches: `feat/<name>`, `fix/<name>`.
- Commits: Conventional Commits.
- PRs: squash-merge to `main`.

## Boundaries

**ALWAYS**

- Use absolute paths in agent responses.
- Respect pre-commit hooks and fix their root causes.
- Build and test after config-related changes.
- Run `go build` before committing.

**USUALLY / ASK FIRST**

- Add new dependencies.
- Modify config format.
- Change command signatures.
- Delete files.

**NEVER**

- Commit secrets or API keys.
- Modify shared user config without approval.
- Bypass or disable quality gates.
