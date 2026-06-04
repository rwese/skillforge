# PRD: skillforge Documentation Refresh

**Version:** 1.0-draft
**Date:** 2026-04-28
**Status:** Draft
**Target:** v1.1.0 (alongside Stability & Quality PRD)

---

## 1. Executive Summary

Improve documentation for skillforge to help new users get started quickly and help contributors understand the codebase.

**Goal:** Comprehensive but not overwhelming. "Docs that a junior dev can understand."

---

## 2. Scope

### In Scope
- README.md refresh with troubleshooting section
- CONTRIBUTING.md for developers
- CHANGELOG.md structure
- TODO.md cleanup

### Out of Scope
- man pages (future)
- API documentation (no public API)
- Video tutorials
- Website

---

## 3. README.md Improvements

### 3.1 Current State

Existing README has:
- Basic installation
- Quick start commands
- Command reference
- Configuration example
- Shell completion

### 3.2 Proposed Structure

```markdown
# skillforge

> A focused CLI for managing agent skills from git repositories.

## Features
- bullet points

## Installation
- go install
- build from source

## Quick Start (5 minutes)
1. Add a skill repository
2. Search for skills
3. Install a skill
4. List installed skills

## Commands
- repo (grouped)
- skill (grouped)
- target (grouped)

## Configuration
- Global vs local
- Config file locations
- Example config

## Troubleshooting      <-- NEW
## Contributing         <-- NEW
## License
```

### 3.3 Troubleshooting Section

```markdown
## Troubleshooting

### "skill not found" error

Run `skillforge skill search <name>` to find available skills.

### "target not found" error

Run `skillforge target list` to see configured targets.

### "no local config found" error

Use `--global` flag to use global config only, or create a local config:

```bash
skillforge target add myagent ~/.myagent/skills/
```

### Git authentication failures

Ensure SSH keys are configured for git remotes, or use HTTPS URLs:

```bash
skillforge repo add https://github.com/user/skills
```

### Config file locations

- **Global:** `~/.config/skillforge/config.toml`
- **Local:** `.skillforge/config.toml` (in git repo or cwd)

Use `--global` or `--local` to force a specific config.

### Verbose output for debugging

Use `-v` flag to see debug information:

```bash
skillforge -v skill list
```
```

### 3.4 Quick Start Examples

```bash
# 1. Add a skill repository
skillforge repo add https://github.com/rwese/agent-skills

# 2. Search for skills
skillforge skill search docker

# 3. Install a skill
skillforge skill install docker -t pi

# 4. List installed skills
skillforge skill list

# 5. Check sync state
skillforge sync --diff

# 6. Apply sync fixes
skillforge sync --fix-all
```

---

## 4. CONTRIBUTING.md

### 4.1 Purpose

Guide new contributors through:
- Development setup
- Code standards
- Testing
- PR process

### 4.2 Proposed Structure

```markdown
# Contributing to skillforge

Thanks for your interest!

## Development Setup

### Prerequisites
- Go 1.21+
- Git

### Clone and Build

```bash
git clone https://github.com/rwese/skillforge
cd skillforge
go build -o skillforge ./cmd/skillforge/
```

### Running Tests

```bash
go test ./... -v
```

### Code Standards

- Run `go fmt` before committing
- Run `go vet` to check for issues
- Keep functions small and focused
- Add tests for new functionality

## Project Structure

```
skillforge/
├── cmd/skillforge/     # CLI commands (cobra)
├── internal/           # Internal packages
│   ├── config/        # Configuration
│   ├── repo/          # Git repository operations
│   └── search/        # Skill search
└── pkg/grimoire/       # Grimoire types
```

## Submitting Changes

1. Fork the repository
2. Create a branch: `git checkout -b feat/my-feature`
3. Make changes with tests
4. Run `go build` and `go test ./...`
5. Commit using Conventional Commits
6. Push and open a PR

## Commit Messages

Use Conventional Commits:

```
feat: add --dry-run flag
fix: clone branch logic for non-main branches
docs: add troubleshooting section
test: add config loading tests
```

## Getting Help

Open an issue for bugs or feature requests.
```

---

## 5. CHANGELOG.md

### 5.1 Structure

```markdown
# Changelog

All notable changes are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Added
### Changed
### Deprecated
### Removed
### Fixed
### Security

## [1.0.0] - 2026-04-27

Initial release.
```

### 5.2 Migration Notes

Include upgrade notes when relevant:

```markdown
## [1.1.0] - 2026-XX-XX

### Added
- `--dry-run` flag for preview mode

### Changed
- Confirmation prompts now required for destructive actions (use `-y` to skip)

### Fixed
- Clone now works correctly with non-main branches
```

---

## 6. TODO.md Cleanup

Current TODO.md tracks Phase 1-3. After v1.1:

```markdown
# skillforge TODO

## v1.1 (Current)
- [x] Bug fixes (clone, ensureTargetDir, scopeLocal)
- [x] Testing coverage
- [x] UX polish (flags, confirmations, errors)

## v2 Future
See `docs/prds/skillforge-v2/README.md`

- [ ] Interactive TUI
- [ ] Suggest command
- [ ] Dirty state detection
- [ ] Doctor command
```

---

## 7. Implementation Checklist

- [ ] Update README.md with troubleshooting section
- [ ] Add quick start examples
- [ ] Create CONTRIBUTING.md
- [ ] Create CHANGELOG.md with structure
- [ ] Clean up TODO.md
- [ ] Verify all links work
- [ ] Test all command examples in README

---

## 8. Files to Change

| File | Action |
|------|--------|
| `README.md` | Rewrite with new sections |
| `CONTRIBUTING.md` | Create new |
| `CHANGELOG.md` | Create new |
| `TODO.md` | Update |

---

## 9. Success Criteria

- [ ] README.md has troubleshooting section with 5+ common issues
- [ ] README.md quick start works in < 5 minutes
- [ ] CONTRIBUTING.md covers setup, testing, PR process
- [ ] CHANGELOG.md follows Keep a Changelog format
- [ ] TODO.md accurately reflects project state
- [ ] All code examples in docs tested manually
