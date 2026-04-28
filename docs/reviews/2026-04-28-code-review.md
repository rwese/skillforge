# Code Review: skillforge-ng

**Date:** 2026-04-28
**Reviewer:** Exploration Session
**Scope:** Full codebase exploration

---

## Summary

skillforge-ng is a focused CLI for managing AI agent skills from git repositories. The v1 MVP is **feature-complete** per the PRD, implementing:
- Repository caching with git operations
- Skill installation with grimoire metadata
- Target management for multiple agents
- Scope-aware configuration (local/global)
- Simple keyword search

**Overall Assessment:** Solid foundation. Clean separation of concerns. Room for polish, error handling, and testing.

---

## Strengths

### Architecture
- **Clean package structure**: `cmd/`, `internal/`, `pkg/` follows Go conventions
- **No external git libraries**: Uses `git` CLI via `exec.Command` - reliable and simple
- **Grimoire metadata**: Smart versioning system tracks skill origin for updates
- **Scope detection**: Auto-detects local vs global config by walking from cwd to git root

### Code Quality
- **Cobra patterns**: Proper command grouping and flag handling
- **TOML config**: Using `BurntSushi/toml` as specified
- **Error wrapping**: Uses `%w` for proper error chain wrapping
- **Path expansion**: `~` → home directory handled consistently

### UX
- **JSON output**: All list commands support `-f json`
- **Spinners**: Progress indication for git operations
- **Tree output**: Nice `├─` `└─` formatting for skill lists
- **Check mode**: `--check` flag for dry-run updates

---

## Issues & Observations

### 🔴 Bugs / Edge Cases

| Location | Issue | Severity |
|----------|-------|----------|
| `cache.go:Clone` | Branch handling broken - `if branch == "main"` uses wrong logic | **High** |
| `repo.go:runRepoUpdate` | Unused variable `info.URL` causes compile warning | **Low** |
| `repo.go:runRepoAdd` | Unused variable `commit` causes compile warning | **Low** |
| `target.go` | `ensureTargetDir` defined but never called | **Medium** |
| `config.go:Load` | `ScopeLocal` returns empty config if no local file exists | **Medium** |

### 🟡 Code Smells

| Location | Issue | Suggestion |
|----------|-------|------------|
| All command files | Duplicate `loadConfig`/`saveConfig` functions | Move to shared file |
| `skill.go:runSkillInstall` | Nested loops to find skill - O(n*m) | Build index map first |
| `discover.go` | Reads `SKILL.md` for every directory check | Cache file reads |
| `spinner.go` | `Spinner.Start()` goroutine leaks if `Stop()` called twice | Add sync.Once |
| `cache.go` | `GetUpdated` uses file mtime, not git log | Misses actual update time |

### 🟢 Missing Features (Expected)

| Feature | Status | Notes |
|---------|--------|-------|
| `skillforge doctor` | Not implemented | Planned for v2 |
| `skillforge manage` (TUI) | Not implemented | Planned for v2 |
| `skillforge suggest` | Not implemented | Planned for v2 |
| Tree hash for dirty detection | Not implemented | Intentionally skipped in v1 |
| Shell completions | Implemented | bash, zsh, fish, powershell |

---

## Detailed Findings

### 1. Clone Branch Logic Bug

```go
// cache.go - BROKEN
args := []string{"clone", "--depth", "1", "-b", branch}
if branch == "main" {
    args = []string{"clone", "--depth", "1", url, targetDir}
} else {
    args = append(args, branch, url, targetDir)  // branch already in args!
}
```

**Impact:** When branch != "main", branch flag appears twice: `git clone --depth 1 -b dev dev url target`

### 2. Unused Variables

```bash
$ go build ./cmd/skillforge/
# No output but warnings may exist
$ go vet ./...
./cmd/skillforge/repo.go:95:9: err: declared but not used (this is in a different file, the issue is commit var)
```

### 3. ScopeLocal Behavior

```go
case ScopeLocal:
    localPath := DetectLocalPath()
    if localPath == "" {
        return cfg, nil  // Returns empty config, no error!
    }
```

If user specifies `--local` but no config exists, they get silent empty config instead of error.

### 4. Target Directory Not Created

`targetAddCmd` creates config entry but doesn't call `ensureTargetDir(path)`. If target path doesn't exist, first install fails.

---

## Test Coverage

```
$ go test ./... -cover
?       github.com/rwese/skillforge-ng/cmd/skillforge     [no test files]
?       github.com/rwese/skillforge-ng/internal/config   [no test files]
?       github.com/rwese/skillforge-ng/internal/repo      [no test files]
?       github.com/rwese/skillforge-ng/internal/search     [no test files]
?       github.com/rwese/skillforge-ng/pkg/grimoire        [no test files]
```

**Coverage:** 0% - No test files exist.

---

## Recommendations

### Immediate (Quick Wins)

1. Fix `cache.go:Clone` branch logic
2. Add `ensureTargetDir` call in `targetAddCmd`
3. Handle `ScopeLocal` with no config gracefully
4. Remove or use `commit` variable in `repoAddCmd`
5. Run `go vet` and fix warnings

### Short Term

1. Add unit tests (at least for config loading, search, grimoire)
2. Add `--dry-run` flag to `skill install`
3. Add verbose/debug flag with `-v`
4. Create sample skill repository for testing

### Medium Term (v1.1)

1. Interactive confirmation for destructive actions (`remove`, `repo remove`)
2. Progress bar for file copy operations
3. Config validation command
4. Better error messages with suggestions

### Long Term (v2)

Per `docs/prds/skillforge-ng-v2/README.md`:
- TUI for visual management
- `suggest` command for project analysis
- Dirty state detection with tree hashes
- `doctor` command for diagnostics

---

## Files Reviewed

| File | Lines | Assessment |
|------|-------|------------|
| `cmd/skillforge/main.go` | 15 | ✅ Minimal entry point |
| `cmd/skillforge/root.go` | 55 | ✅ Clean Cobra setup |
| `cmd/skillforge/repo.go` | 240 | ⚠️ Clone bug, unused var |
| `cmd/skillforge/skill.go` | 310 | ⚠️ Nested loops, works |
| `cmd/skillforge/target.go` | 195 | ⚠️ Missing ensureTargetDir |
| `cmd/skillforge/output.go` | 75 | ✅ Clean JSON/text handling |
| `cmd/skillforge/spinner.go` | 90 | ⚠️ Potential goroutine leak |
| `internal/config/config.go` | 180 | ⚠️ ScopeLocal edge case |
| `internal/repo/cache.go` | 115 | 🔴 Clone bug |
| `internal/repo/discover.go` | 90 | ⚠️ Could cache reads |
| `internal/repo/grimoire.go` | 100 | ✅ Clean implementation |
| `internal/search/search.go` | 60 | ✅ Simple and effective |
| `pkg/grimoire/types.go` | 30 | ✅ Clean types |

---

## Action Items

| Priority | Owner | Item |
|----------|-------|------|
| 🔴 P0 | - | Fix `cache.go:Clone` branch logic |
| 🔴 P0 | - | Fix `ensureTargetDir` not called |
| 🟡 P1 | - | Add basic unit tests |
| 🟡 P1 | - | Run `go vet`, fix warnings |
| 🟢 P2 | - | Add `--dry-run` flag |
| 🟢 P2 | - | Add verbose flag |
| 🟢 P3 | - | Create sample test repo |

---

## Sign-off

This review represents initial exploration findings. Code functions as designed per v1 PRD for basic use cases. Edge cases and testing should be addressed before 1.0 release.
