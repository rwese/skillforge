# PRD: skillforge Usability Improvements

## Project Name: skillforge

**Version:** 1.0.0 **Date:** 2026-04-28 **Status:** Approved **Target:** skillforge v0.2.0

---

### 1. Executive Summary

**Problem Statement**

After a junior developer review session, several usability and correctness issues were identified in skillforge:

1. **Scope auto-detection bug**: `target list` without flags doesn't show targets when global config exists but local config doesn't
2. **Poor onboarding**: New users get cryptic errors like "no enabled targets found" without guidance
3. **Inconsistent output**: Search results show `---` for missing descriptions, list output lacks formatting

**Related Projects**

| Project | Purpose | Tech | Status |
|---------|---------|------|--------|
| skillforge | Main CLI tool | Go, Cobra | Active |

**Proposed Solution**

Implement targeted usability fixes based on review feedback:
- Fix config scope merging (key-level merge: local overrides global for same keys)
- Add onboarding hints to error messages (always shown)
- Improve output formatting (colors, tables, compact mode)

---

### 2. Goals & Non-Goals

**Goals**

1. Fix config scope auto-detection to merge global + local configs
2. Add helpful onboarding hints to all error messages (always shown)
3. Implement colored output for better readability (auto-detect terminal)
4. Add table formatting for list commands (all columns in default)
5. Implement compact output mode (TARGET/NAME@COMMIT format)
6. Implement auto-switch to compact when piped
7. Write unit tests for config merging behavior
8. Write integration tests for CLI commands

**Non-Goals**

- Not changing the config file format
- Not adding interactive prompts (keep CLI scriptable)
- Not adding a GUI or TUI
- Not changing the skill discovery logic
- Not adding `--wide` flag (table is complete by default)
- Not adding `--format=csv` support
- Not adding `HINTS` environment variable (hints always on)
- Not adding debug hints to verbose mode

---

### 3. Technical Architecture

**Technology Stack**

| Layer | Technology | Rationale |
|-------|------------|-----------|
| Language | Go 1.21+ | Already in use |
| CLI | Cobra | Already in use |
| Config | TOML (BurntSushi) | Already in use |
| Testing | Go test | Already in use |
| Output | lipgloss | Lightweight terminal styling |

**Project Structure**

```
skillforge/
├── cmd/skillforge/
│   ├── main.go
│   ├── root.go           # Flags, config loading
│   ├── output.go         # NEW: Output formatting
│   ├── color.go          # NEW: Color definitions
│   ├── skill.go
│   ├── target.go
│   └── repo.go
├── internal/
│   ├── config/
│   │   ├── config.go     # MODIFY: Fix scope merging
│   │   └── config_test.go # MODIFY: Add merging tests
│   ├── repo/
│   └── search/
├── pkg/grimoire/
└── docs/prds/
    └── skillforge-usability-improvements/
        └── README.md     # This document
```

**Configuration**

Config merging behavior:

```bash
1. Load global config (~/.config/skillforge/config.toml)
2. Load local config (.skillforge/config.toml)
3. Key-level merge:
   - Same keys: local values override global
   - Different keys: merge both
4. Defaults: Use hardcoded defaults if no config exists
```

**Example Merge**

```toml
# Global config
[cache]
path = "~/.cache/skillforge/repos"

[targets.pi]
path = "~/.pi/agent/skills/"
enabled = true

# Local config (.skillforge/config.toml)
[targets.local]
path = "./.pi/skills/"
enabled = true

# Merged result
[cache]
path = "~/.cache/skillforge/repos"

[targets.pi]
path = "~/.pi/agent/skills/"
enabled = true

[targets.local]
path = "./.pi/skills/"
enabled = true
```

---

### 4. Interface Specification

**Global Flags**

| Flag | Description |
|------|-------------|
| `-n, --dry-run` | Preview without applying |
| `-y, --yes` | Skip confirmations |
| `-v, --verbose` | Debug output (no extra hints) |
| `-g, --global` | Use global config only |
| `-l, --local` | Use local config only |
| `--config string` | Custom config path |
| `-f, --format string` | Output format: text, json, compact (default: text) |

**Color Output**

Auto-detected based on terminal:
- Check `NO_COLOR` environment variable
- Check `TERM` environment variable
- Check if stdout is a terminal (isatty)

```bash
✓ Green checkmarks for success
✗ Red X for errors
⚠ Yellow warning for conflicts
→ Cyan arrows for redirects/actions
```

**Auto-Compact Mode**

When stdout is not a terminal (piped/redirected):
- Automatically switches to compact format
- Override with explicit `-f text`

**Onboarding Hints** (always shown in errors)

```bash
Error: no enabled targets found.

Hint:
  • Run: skillforge target add <name> <path> -e
  • Example: skillforge target add pi ~/.pi/agent/skills/ -e
```

**Table Output** (`-f table`)

| TARGET | SKILL | COMMIT | SOURCE | INSTALLED |
|--------|-------|--------|--------|-----------|
| pi | docker | a0c036f | github.com/rwese/agents-grimoire | 2026-04-28 |
| pi | git | b1d2e3f | github.com/rwese/agents-grimoire | 2026-04-28 |

**Compact Output** (`-f compact` or auto-piped)

```
pi/docker@a0c036f
pi/git@b1d2e3f
```

Format: `TARGET/NAME@COMMIT`

**JSON Output** (`-f json`)

```json
{
  "skills": [
    {
      "name": "docker",
      "commit": "a0c036f",
      "target": "pi",
      "source": "github.com/rwese/agents-grimoire",
      "description": "Configure and use Docker with local registries"
    }
  ]
}
```

**Output Ordering**

All list outputs are sorted alphabetically by name.

---

### 5. Feature Roadmap

**Implementation Order:** TDD - Write failing tests first

- [ ] **Phase 1: Config Merge Fix**
  - [ ] Write failing unit tests for config merging
  - [ ] Fix `config.Load()` to merge global + local
  - [ ] Fix `config.Save()` to preserve all scopes
  - [ ] Verify all tests pass

- [ ] **Phase 2: Onboarding Hints**
  - [ ] Create hint formatting function
  - [ ] Add hints to `skill install` errors
  - [ ] Add hints to `skill search` errors
  - [ ] Add hints to `repo` command errors
  - [ ] Add hints to `target` command errors

- [ ] **Phase 3: Output Improvements**
  - [ ] Add lipgloss dependency
  - [ ] Create `color.go` with color definitions
  - [ ] Create `output.go` with formatting functions
  - [ ] Implement color auto-detection
  - [ ] Update `skill list` with table/compact/json
  - [ ] Update `target list` with table/compact/json
  - [ ] Update `repo list` with table/compact/json
  - [ ] Implement auto-compact for pipes

- [ ] **Phase 4: Integration Tests**
  - [ ] Set up integration test environment (temp cwd + local config)
  - [ ] Test full `skill install` flow
  - [ ] Test `skill remove` flow
  - [ ] Test `target add/list/remove` cycle
  - [ ] Test scope merging with temp directories
  - [ ] Test output formats (text/table/json/compact)

**Milestone:** Phase 1-2 are MVP (core usability fixes). Phase 3-4 are polish.

---

### 6. Integration Notes

**Breaking Changes**

| Aspect | Current | New |
|--------|---------|-----|
| Config format | TOML | TOML (unchanged) |
| Config location | ~/.config/skillforge/ | ~/.config/skillforge/ (unchanged) |
| CLI flags | All preserved | All preserved + new `-f` flag |
| Output (terminal) | Plain text | Plain text (default unchanged) |
| Output (piped) | Plain text | Compact (auto) |

**Migration Path**

- Configs: No migration needed
- Piped scripts: May need `-f text` if relying on old format
- JSON output: Added `description` field to skills

---

### 7. Decisions Log

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Colors default | Auto-detect terminal | User-friendly, respects automation |
| Compact format | TARGET/NAME@COMMIT | Clearer grouping |
| Hints control | Always shown | Maximum discoverability |
| Auto-compact | Yes, piped | Sensible script defaults |
| Config merge | Key-level | Granular control |
| Empty config | Hardcoded defaults | Don't pollute filesystem |
| Table columns | All (no --wide) | Complete by default |
| Hint format | Separate section | Clear separation |
| Confirm prompts | No hints | Keep minimal |
| TERM check | Yes | Respect user terminal settings |

---

### 8. Success Criteria

- [ ] `skillforge target list` shows targets from global config without `-g` flag
- [ ] Local config adds targets without removing global targets
- [ ] All error messages include actionable hints
- [ ] Hints display in separate `Hint:` section with bullet points
- [ ] `skill list -f table` produces aligned columns: TARGET, SKILL, COMMIT, SOURCE, INSTALLED
- [ ] `skill list -f compact` produces `TARGET/NAME@COMMIT` format
- [ ] `skill list` output is sorted alphabetically
- [ ] Colored output displays correctly in terminals
- [ ] `NO_COLOR=1` disables colors
- [ ] `TERM=dumb` disables colors
- [ ] Piping output auto-uses compact format
- [ ] Explicit `-f text` overrides auto-compact
- [ ] Unit tests cover all config merge scenarios
- [ ] Integration tests cover main command flows
- [ ] All existing tests continue to pass

---

### 9. Repository Location

```bash
/Users/wese/Repos/github.com/rwese/skillforge/
```

---

### Appendix: Example Sessions

**New User Journey (After Fixes)**

```bash
$ skillforge skill install docker
Error: no enabled targets found.

Hint:
  • Run: skillforge target add <name> <path> -e
  • Example: skillforge target add pi ~/.pi/agent/skills/ -e

$ skillforge target add pi ~/.pi/agent/skills/ -e
✓ Added target pi

$ skillforge skill install docker
Installing docker to pi...
  ✓ docker installed to pi

$ skillforge skill list -f table
TARGET    SKILL       COMMIT     SOURCE                           INSTALLED
────────────────────────────────────────────────────────────────────────────────
pi        docker      a0c036f    github.com/rwese/agents-grimoire 2026-04-28

$ skillforge skill list -f compact
pi/docker@a0c036f

$ skillforge skill list | cat
pi/docker@a0c036f

$ skillforge skill search postgres
No skills found matching "postgres".

Hint:
  • Run: skillforge repo update to refresh skills

$ skillforge skill list -f json
{
  "skills": [
    {
      "name": "docker",
      "commit": "a0c036f",
      "target": "pi",
      "source": "github.com/rwese/agents-grimoire",
      "description": "Configure and use Docker with local registries"
    }
  ]
}
```

**Colored Output**

```bash
$ skillforge skill install docker
Installing docker to pi...
  ✓ docker installed to pi        # Green ✓

$ skillforge skill install docker
  ! docker already exists         # Yellow warning

$ skillforge skill install nonexistent
Error: skill "nonexistent" not found.   # Red Error

Hint:
  • Run: skillforge skill search <query>
```

**No Color Mode**

```bash
$ NO_COLOR=1 skillforge skill install docker
Installing docker to pi...
  ✓ docker installed to pi

$ TERM=dumb skillforge skill list
TARGET    SKILL       COMMIT
pi        docker      a0c036f
```

---

### Appendix: Dependencies

```go
// go.mod additions
github.com/charmacel/lipgloss v0.9.0
```

---

### Appendix: Test Plan

**Unit Tests** (`internal/config/config_test.go`)

| Test | Description | Expected |
|------|-------------|----------|
| `TestLoad_GlobalOnly` | Only global config exists | Returns global values |
| `TestLoad_LocalOnly` | Only local config exists | Returns local values |
| `TestLoad_Neither` | No config exists | Returns defaults |
| `TestLoad_BothMerge` | Both exist | Merges all keys |
| `TestLoad_LocalOverrides` | Same key in both | Local value wins |
| `TestLoad_DifferentKeys` | Different keys in each | Both values present |
| `TestSave_LocalPreservesGlobal` | Save local | Global remains intact |
| `TestSave_GlobalPreservesLocal` | Save global | Local remains intact |

**Integration Tests** (`cmd/skillforge/integration_test.go`)

Setup: Use temp directories with `--config` flag

| Test | Description |
|------|-------------|
| `TestSkillInstall_FullFlow` | Add repo → search → install → list |
| `TestSkillRemove_Flow` | Install → remove → verify gone |
| `TestTargetManagement` | add → list → remove cycle |
| `TestScopeInheritance` | Create local, verify global still works |
| `TestOutputFormats` | Verify text/table/json/compact all work |
| `TestAutoCompact` | Verify piped output uses compact |
| `TestNoColorEnv` | Verify NO_COLOR disables colors |
| `TestHintsInErrors` | Verify hints appear in all error cases |

---

### Appendix: Hints Reference

| Command | Error | Hint |
|---------|-------|------|
| `skill install` | no targets | Run: skillforge target add |
| `skill install` | no repos | Run: skillforge repo add |
| `skill install` | not found | Run: skillforge repo update |
| `skill search` | no repos | Run: skillforge repo add |
| `skill search` | no results | Run: skillforge repo update |
| `skill remove` | not found | Skill not installed |
| `target add` | path exists | Use different name or path |
| `repo add` | exists | Run: skillforge repo update |
| `repo remove` | not found | Repository not cached |
