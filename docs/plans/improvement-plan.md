# Improvement Plan: skillforge-ng

**Date:** 2026-04-28
**Status:** Planning
**Target Version:** v1.1

---

## Overview

Based on the [code review](./reviews/2026-04-28-code-review.md), this plan addresses critical bugs, missing tests, and UX improvements for skillforge-ng.

---

## Release Phases

### Phase 1: Bug Fixes (v1.0.1)
**Goal:** Ship fixes without new features

| # | Issue | File | Fix | Effort |
|---|-------|------|-----|--------|
| 1 | Clone branch logic broken | `internal/repo/cache.go` | Rewrite args construction | 15m |
| 2 | `ensureTargetDir` never called | `cmd/skillforge/target.go` | Add call after target add | 10m |
| 3 | ScopeLocal returns empty config | `internal/config/config.go` | Return error or fallback | 15m |
| 4 | Unused `commit` variable | `cmd/skillforge/repo.go` | Remove or use | 5m |
| 5 | Spinner goroutine leak | `cmd/skillforge/spinner.go` | Add sync.Once | 20m |

**Verification:**
```bash
go build ./... && go vet ./... && ./skillforge repo add --help
```

---

### Phase 2: Testing (v1.0.2)
**Goal:** Add test coverage for core functionality

| Package | Tests Needed | Coverage Target |
|---------|--------------|-----------------|
| `internal/config` | Load, Save, ExpandPath, Scope detection | 80% |
| `internal/search` | Keyword matching, ranking | 90% |
| `pkg/grimoire` | Read/Write grimoire, types | 80% |
| `internal/repo` | Cache operations (mock git) | 60% |

**Test Structure:**
```
internal/
├── config/
│   ├── config_test.go
│   └── testdata/
│       ├── global.toml
│       └── local.toml
├── search/
│   └── search_test.go
└── repo/
    └── repo_test.go  # Mock git commands

pkg/
└── grimoire/
    ├── grimoire_test.go
    └── testdata/
        └── valid.grimoire
```

**Run Tests:**
```bash
go test ./... -v -cover
```

---

### Phase 3: UX Polish (v1.1.0)
**Goal:** Improve user experience without core changes

#### 3.1 New Flags

| Command | Flag | Description | Example |
|---------|------|-------------|---------|
| `skill install` | `--dry-run` | Preview without installing | `skill install docker --dry-run` |
| `skill install` | `-y` | Skip confirmation prompts | `skill install docker -y` |
| Global | `-v, --verbose` | Enable debug output | `skillforge -v skill list` |
| Global | `--no-color` | Disable colored output | `skillforge --no-color list` |

#### 3.2 Confirmation Prompts

Add interactive confirmation for destructive operations:

```go
// Before: silent removal
skill repo remove <name>

// After: confirmation prompt
$ skillforge repo remove myrepo
? Remove cached repository "myrepo" and all installed skills from targets? [y/N]
```

**Implementation:** Use a simple yes/no prompt (no external dialog lib).

#### 3.3 Better Error Messages

| Before | After |
|--------|-------|
| `skill not found` | `Skill "docker" not found. Run 'skillforge skill search docker' to find available skills.` |
| `target not found` | `Target "pi" not found. Run 'skillforge target list' to see available targets.` |
| `config not found` | `No config found. Run 'skillforge target add pi ~/.pi/agent/skills/' to create one.` |

#### 3.4 Progress for File Operations

Currently only git operations have spinners. Add progress for skill copy:

```go
// Use existing Progress struct from spinner.go
p := NewProgress(fileCount, "Installing skill")
for i, file := range files {
    p.SetFile(file)
    copyFile(src, dst)
    p.Increment()
}
p.Complete()
```

---

### Phase 4: Documentation (v1.1.0)
**Goal:** Improve user and developer docs

| Doc | Status | Action |
|-----|--------|--------|
| README.md | ✅ Basic | Add troubleshooting section |
| man page | ❌ Missing | Generate with cobra-doc |
| Examples | ❌ Minimal | Add common workflows |
| Contributing | ❌ Missing | Add CONTRIBUTING.md |

#### Troubleshooting Section (README.md)

```markdown
## Troubleshooting

### "skill not found" error
Run `skillforge skill search <name>` to find available skills.

### "target not found" error
Run `skillforge target list` to see configured targets.

### Config not loading
- Check config location: `~/.config/skillforge/config.toml` (global) or `.skillforge/config.toml` (local)
- Use `--global` or `--local` to force scope

### Git authentication failures
Ensure SSH keys are configured for git remotes, or use HTTPS URLs.
```

---

### Phase 5: CLI Enhancements (v1.2.0)
**Goal:** Add commonly requested CLI features

#### 5.1 Config Validation Command

```bash
$ skillforge config validate
✓ Global config valid (~/.config/skillforge/config.toml)
✓ Local config valid (./.skillforge/config.toml)
✓ 3 targets configured (2 enabled)
✓ 2 repositories cached (15 skills)
✓ No issues found
```

#### 5.2 Import/Export

```bash
# Export installed skills to file
skillforge config export > skills-backup.toml

# Import from file
skillforge config import skills-backup.toml
```

#### 5.3 Quick Install from URL

```bash
# Install skill directly from git URL (creates temporary repo)
skillforge skill install https://github.com/user/skill-repo/blob/main/skills/docker
```

#### 5.4 Target Permissions Check

```bash
$ skillforge target list
  pi        ~/.pi/agent/skills/    (enabled, writable)
  claude    ~/.claude/skills/       (enabled, NOT WRITABLE ⚠)
```

---

## Backlog (v2)

Per [v2 PRD](../prds/skillforge-ng-v2/README.md):

| Feature | Priority | Dependencies |
|---------|----------|--------------|
| TUI (`manage`) | P1 | bubble/lipgloss |
| `doctor` command | P1 | tree hash tracking |
| `suggest` command | P2 | project analysis |
| Dirty state detection | P2 | tree hash |
| Auto-completion for skill names | P3 | - |

---

## Task Board

### Ready to Start

- [ ] Fix cache.go Clone branch logic
- [ ] Add ensureTargetDir call in target.go
- [ ] Run go vet, fix warnings
- [ ] Create config_test.go
- [ ] Create search_test.go

### In Progress

_(None)_

### Done

- [x] Code exploration
- [x] Write code review document
- [x] Create improvement plan

---

## Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Test coverage | 0% | 60% (v1.0.2) |
| Go vet warnings | 4+ | 0 |
| CLI flags | 8 | 12+ |
| Documentation pages | 2 | 4+ |

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-------------|--------|------------|
| Breaking config format | Low | High | Version config, provide migration |
| Test failures on CI | Medium | Medium | Mock git operations |
| Scope detection edge cases | Medium | Medium | Add integration tests |

---

## Timeline

```
Week 1: Phase 1 (Bug Fixes)
  - Fix 5 bugs
  - Verify with go build + go vet

Week 2-3: Phase 2 (Testing)
  - Write unit tests for config, search, grimoire
  - Add integration test helpers

Week 4: Phase 3 (UX Polish)
  - Add --dry-run, --verbose flags
  - Confirmation prompts
  - Better error messages

Week 5-6: Phase 4 (Documentation)
  - Expand README
  - Add CONTRIBUTING.md
  - Generate man page

Week 7+: Phase 5 (CLI Enhancements)
  - config validate
  - Import/export
```

---

## Getting Started

1. **Review** the [code review](../reviews/2026-04-28-code-review.md)
2. **Pick** a task from "Ready to Start"
3. **Create** a branch: `fix/<name>` or `feat/<name>`
4. **Implement** with tests
5. **Submit** PR with description referencing this plan
