# PRD: skillforge-ng v1.1 - Stability & Quality

**Version:** 1.1.0-draft
**Date:** 2026-04-28
**Status:** Draft
**Target:** v1.1.0

---

## 1. Executive Summary

This PRD covers three related improvements to skillforge-ng:

1. **Bug Fixes (v1.0.1)** - Critical issues identified in code review
2. **Testing (v1.0.2)** - Unit and integration test coverage
3. **UX Polish (v1.1.0)** - Flags, confirmations, verbosity, error messages

**Goal:** Ship a stable, well-tested v1.1 with excellent UX while keeping the codebase simple.

---

## 2. Scope Boundaries

### In Scope
- Fix critical bugs from code review
- Add unit + integration tests
- Add `--dry-run`, `-y/--yes`, `-v/--verbose` flags
- Confirmation prompts for destructive operations
- Better error messages with context
- Progress indication for file operations
- Standard library testing only (no external deps)

### Out of Scope
- TUI (`skillforge manage`)
- `doctor` command
- `suggest` command
- `config validate` command (minimal config check only)
- `import`/`export` functionality
- Shell completions (already done)
- Config format changes
- New dependencies

---

## 3. Bug Fixes (v1.0.1)

### 3.1 Clone Branch Logic (Critical)

**File:** `internal/repo/cache.go`

**Issue:** Branch flag appears twice when branch != "main"

```go
// BROKEN
args := []string{"clone", "--depth", "1", "-b", branch}
if branch == "main" {
    args = []string{"clone", "--depth", "1", url, targetDir}
} else {
    args = append(args, branch, url, targetDir)  // branch already in args!
}
```

**Fix:**
```go
var args []string
if branch == "main" {
    args = []string{"clone", "--depth", "1", url, targetDir}
} else {
    args = []string{"clone", "--depth", "1", "-b", branch, url, targetDir}
}
```

### 3.2 Target Directory Creation (Critical)

**File:** `cmd/skillforge/target.go`

**Issue:** `ensureTargetDir()` defined but never called

**Fix:** Call `ensureTargetDir(path)` after adding target to config.

### 3.3 ScopeLocal Behavior (Medium)

**File:** `internal/config/config.go`

**Issue:** `--local` flag silently returns empty config if no local file exists

**Fix:**
```go
case ScopeLocal:
    localPath := DetectLocalPath()
    if localPath == "" {
        return nil, fmt.Errorf("no local config found in .skillforge/")
    }
    if err := l.loadFile(localPath, cfg); err != nil {
        return nil, err
    }
```

### 3.4 Unused Variables

**Files:** `cmd/skillforge/repo.go`

**Fix:** Remove unused `commit` variable or use `_ = commit` to silence.

### 3.5 Spinner Goroutine Leak (Low)

**File:** `cmd/skillforge/spinner.go`

**Issue:** `Stop()` called twice causes panic

**Fix:** Use `sync.Once` or select with default case.

---

## 4. Testing (v1.0.2)

### 4.1 Philosophy

- **Standard library only**: `testing`, `os`, `os/exec`, `io/ioutil`
- **No external deps**: Keep `go.sum` clean
- **Real git operations**: Integration tests use temp dirs with actual git

### 4.2 Test Structure

```
internal/
├── config/
│   └── config_test.go        # Config loading, saving, scope detection
├── repo/
│   └── repo_test.go          # Cache operations, skill discovery
├── search/
│   └── search_test.go        # Keyword matching, ranking
└── grimoire/
    └── grimoire_test.go      # Grimoire read/write

cmd/
└── skillforge/
    └── integration_test.go   # E2E via subprocess
```

### 4.3 Coverage Targets

| Package | Target | Notes |
|---------|--------|-------|
| `internal/config` | 80% | Load, Save, ExpandPath, Scope |
| `internal/repo` | 60% | Mock git, real temp dirs |
| `internal/search` | 90% | All ranking cases |
| `pkg/grimoire` | 80% | Read/write/grimoire |

### 4.4 Test Cases

#### Config Tests
- [ ] Load global config
- [ ] Load local config
- [ ] Auto-detect scope (local in cwd, local in git root, global fallback)
- [ ] Merge local + global configs
- [ ] ExpandPath `~` to home
- [ ] ContractPath home to `~`
- [ ] Save to global path
- [ ] Save to local path (create dir if missing)
- [ ] Error on missing local when `--local` specified

#### Search Tests
- [ ] Exact name match scores highest
- [ ] Prefix match scores high
- [ ] Contains match scores medium
- [ ] Description match scores low
- [ ] Multiple word query
- [ ] Case insensitive
- [ ] Empty results for no matches
- [ ] Search across multiple repos

#### Grimoire Tests
- [ ] Write valid grimoire
- [ ] Read grimoire with all fields
- [ ] Read missing grimoire returns nil
- [ ] Invalid TOML returns error

#### Repo Tests (Integration)
- [ ] Clone repository
- [ ] Fetch updates
- [ ] Pull updates
- [ ] Remove cached repo
- [ ] Get current commit
- [ ] Discover skills in repo
- [ ] Install skill to target
- [ ] List installed skills
- [ ] Remove installed skill

### 4.5 Test Helpers

```go
// Use testify? NO - standard library only

// tempDir creates a temporary directory that auto-cleans
func tempDir(t *testing.T) string {
    dir, err := os.MkdirTemp("", "skillforge-test-*")
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() { os.RemoveAll(dir) })
    return dir
}

// gitInit creates a git repo in the given directory
func gitInit(t *testing.T, dir string) {
    // run git init, git config, etc.
}
```

---

## 5. UX Improvements (v1.1.0)

### 5.1 New Flags

#### `--dry-run` Flag

**Commands:** `skill install`, `skill remove`, `repo remove`, `target remove`

**Behavior:** Preview what would happen without making changes

```bash
$ skillforge skill install docker --dry-run
[DRY RUN] Would install "docker" to target "pi"
[DRY RUN]   Source: ~/.cache/skillforge/repos/github.com/skills
[DRY RUN]   Target: ~/.pi/agent/skills/docker
[DRY RUN]   Commit: abc1234

$ skillforge skill install docker
Installing docker to pi...
  ✓ docker installed to pi
```

#### `-y, --yes` Flag

**Commands:** `skill install`, `skill remove`, `repo remove`, `target remove`

**Behavior:** Skip confirmation prompts

```bash
$ skillforge repo remove myrepo -y
✓ Removed myrepo
```

#### `-v, --verbose` Flag

**Scope:** Global (all commands)

**Behavior:** Show debug information

```bash
$ skillforge -v skill list
[DEBUG] Loading config from ~/.config/skillforge/config.toml
[DEBUG] Config loaded: 2 targets, 1 repo
[DEBUG] Target "pi": path=~/.pi/agent/skills/, enabled=true
[DEBUG] Scanning ~/.pi/agent/skills/...
Installed Skills:
  pi/
    └─ github  @ abc1234

[DEBUG] Total skills: 1
```

### 5.2 Confirmation Prompts

**Commands:** `repo remove`, `skill remove`, `target remove`

**Behavior:** Interactive confirmation by default

```bash
$ skillforge repo remove myrepo
? Remove cached repository "myrepo"? [y/N]
```

```bash
$ skillforge skill remove docker
? Remove "docker" from target "pi"? [y/N]
```

```bash
$ skillforge target remove pi
? Remove target "pi" and all installed skills? [y/N]
```

**Default:** `N` (safe - user must explicitly confirm)

**Exit codes:**
- 0: Action completed
- 1: Aborted (user said no or error)

### 5.3 Better Error Messages

**Principle:** Errors should include context about what went wrong and what to do.

| Scenario | Before | After |
|----------|--------|-------|
| Skill not found | `skill "x" not found` | `Skill "x" not found in any cached repository. Run 'skillforge skill search x' to find available skills.` |
| Target not found | `target "x" not found` | `Target "x" not found. Run 'skillforge target list' to see configured targets.` |
| Repo not cached | `repo "x" not cached` | `Repository "x" not cached. Run 'skillforge repo add <url>' to cache it.` |
| Config error | `failed to load config` | `Failed to load config: no such file. Run 'skillforge target add <name> <path>' to create a config.` |
| Git error | `failed to clone` | `Failed to clone https://github.com/user/repo: authentication failed. Ensure SSH keys are configured or use HTTPS URL.` |

### 5.4 Progress Indication

**Current:** Spinners for git operations only

**Add:** Progress for skill copy operations

```bash
$ skillforge skill install docker
Installing docker to pi...  [████████████████████] 100% (24 files)
  ✓ docker installed to pi
```

**Implementation:** Extend existing `Progress` struct in `spinner.go`.

---

## 6. Command Changes

### 6.1 Modified Commands

| Command | Changes |
|---------|---------|
| `skill install` | + `--dry-run`, `-y` |
| `skill remove` | + `--dry-run`, `-y`, confirmation |
| `repo remove` | + `--dry-run`, `-y`, confirmation |
| `target remove` | + `--dry-run`, `-y`, confirmation |
| (all) | + `-v, --verbose` |

### 6.2 Flag Summary

| Flag | Scope | Commands |
|------|-------|----------|
| `--dry-run` | command | install, remove (repo/skill/target) |
| `-y, --yes` | command | install, remove (repo/skill/target) |
| `-v, --verbose` | global | all |
| `-g, --global` | global | all (existing) |
| `-l, --local` | global | all (existing) |

---

## 7. Release Plan

### Version Bumps

| Change | Version | Example Tag |
|--------|---------|-------------|
| Bug fixes only | Patch | v1.0.2 |
| New flags (non-breaking) | Minor | v1.1.0 |
| Breaking changes | Major | v2.0.0 |

### Changelog

Manual `CHANGELOG.md` updates per release:

```markdown
## [1.1.0] - 2026-XX-XX

### Added
- `--dry-run` flag for install/remove commands
- `-y/--yes` flag to skip confirmation prompts
- `-v/--verbose` flag for debug output
- Confirmation prompts for destructive actions

### Fixed
- Clone branch logic for non-main branches
- Target directory creation on add
- ScopeLocal now returns error if no config found

### Testing
- Added unit tests for config, search, grimoire packages
- Added integration tests with real git operations
```

### Git Flow

1. Branch from `main`: `git checkout -b fix/stability-v1.1`
2. Implement changes with tests
3. PR to `main`
4. Tag release: `git tag v1.1.0 && git push --tags`

---

## 8. Technical Decisions

| Question | Decision | Rationale |
|----------|----------|-----------|
| Testing lib | Standard library only | No external deps, keep simple |
| Mock git? | Real git in temp dirs | Simpler, tests real behavior |
| Verbosity levels | Just -v flag | Avoid over-engineering |
| Confirm default | No (safe) | Users must explicitly confirm destructive actions |
| Dry-run mode | Preview only | Clear separation, user runs again to execute |

---

## 9. Files to Change

### Bug Fixes
- `internal/repo/cache.go` - Clone logic
- `cmd/skillforge/target.go` - ensureTargetDir call
- `internal/config/config.go` - ScopeLocal error
- `cmd/skillforge/repo.go` - unused var
- `cmd/skillforge/spinner.go` - goroutine leak

### New Flags
- `cmd/skillforge/root.go` - add verbose flag
- `cmd/skillforge/repo.go` - add dry-run, yes flags
- `cmd/skillforge/skill.go` - add dry-run, yes flags
- `cmd/skillforge/target.go` - add dry-run, yes flags

### Confirmation Prompts
- `cmd/skillforge/output.go` - add confirmPrompt function
- Update repo/skill/target remove handlers

### Error Messages
- Update error returns in repo.go, skill.go, target.go
- Add suggestion text where appropriate

### Testing
- `internal/config/config_test.go`
- `internal/repo/repo_test.go`
- `internal/search/search_test.go`
- `pkg/grimoire/grimoire_test.go`
- `cmd/skillforge/integration_test.go`

---

## 10. Success Criteria

- [ ] All 5 bugs fixed and verified
- [ ] Test coverage ≥ 60% overall
- [ ] `go test ./...` passes
- [ ] `go vet ./...` reports 0 warnings
- [ ] `--dry-run` shows preview without changes
- [ ] `-y` skips confirmation prompts
- [ ] `-v` shows debug output
- [ ] Confirmation prompts work for all destructive commands
- [ ] Error messages include context and suggestions
- [ ] CHANGELOG.md updated for v1.1.0
- [ ] Manual testing confirms all new features work

---

## 11. Out of Scope (Future)

- TUI interface
- `doctor` command
- `suggest` command
- `config validate` command
- Import/export
- Auto-update mechanism
- Plugin system
- Complex dirty state detection
