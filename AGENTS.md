# skillforge-ng Agent Instructions

CLI tool for managing agent skills from git repositories.

## Commands

```bash
go build -o skillforge ./cmd/skillforge/  # Build binary
go test ./...                              # Run tests
go mod tidy                                # Update dependencies
```

## Structure

```
skillforge-ng/
├── cmd/skillforge/     # CLI commands (cobra)
├── internal/
│   ├── config/        # Config loading + scope detection
│   ├── repo/          # Git cache, skill discovery, grimoire
│   └── search/        # Simple keyword search
├── pkg/grimoire/       # Grimoire types
└── docs/prds/         # Project documentation
```

## Code Style

- Go 1.21+
- cobra for CLI structure
- TOML for config
- Simple keyword search (no external search libraries)
- No external git libraries (exec git)

## Testing

```bash
go test ./... -v    # Run with verbose output
go test ./... -cover # With coverage
```

## Git Workflow

- Branches: `feat/<name>`, `fix/<name>`
- Commits: Conventional Commits (see skill:git-conventional-commits)
- PRs: squash-merge to main

---

## Boundaries

**ALWAYS**
- Run `go build` before committing
- Test CLI commands manually after major changes
- Update TODO.md when completing tasks

**USUALLY / ASK FIRST**
- Add new dependencies
- Modify config format
- Change command signatures

**NEVER**
- Commit secrets or API keys
- Modify shared config without approval
