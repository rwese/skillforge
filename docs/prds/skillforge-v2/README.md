# PRD: SkillForge-NG v2 - Diagnostics & Dirty State Detection

**Version:** 2.0.0-draft **Date:** 2026-04-28 **Status:** Planned **Target:** v2.0

---

## 1. Executive Summary

Two focused features: a diagnostics command and automatic dirty state detection.

## 2. Planned Features

### 2.1 `skillforge doctor` Command

**Description:** Run diagnostics to validate the skillforge installation.

**Checks:**
| Check | Description |
|-------|-------------|
| Config | Valid TOML, paths exist, targets accessible |
| Cache | Cached repos exist, git remotes reachable |
| Skills | Valid grimoire, source repos accessible |

**Output:**
- Human-readable detailed report
- Exit code: 0 = healthy, non-zero = problems found
- Suggestions for fixing issues

**Example:**
```
$ skillforge doctor
Checking config... OK
  Config path: ~/.config/skillforge/config.toml
  Targets: 2 configured

Checking cache... WARN
  ! repo "skills" not found in cache (run: skillforge repo update)
  * repo "agents" is stale (2 weeks old)

Checking skills... OK
  * 12 skills installed, all healthy

Run 'skillforge repo update' to restore missing cache.
```

### 2.2 Dirty State Detection

**Description:** Detect when installed skills differ from their source.

**Triggers:**
- `skill list` - auto-check installed skills
- `sync --diff` - fetch and report source changes

**Scenarios detected:**
- Source repo updated but skill not re-installed
- Skill manually modified after installation
- Target directory deleted/corrupted

**Implementation:**
- Add `tree_hash` field to grimoire (SHA256 of skill files)
- Compute tree_hash during install
- Compare tree_hash vs source on list/update
- Flag with `↻` indicator in list output

**Example:**
```
$ skillforge skill list
Installed Skills:
  pi/
    ├─ docker-build  @ abc1234
    ├─ git-conventional @ def5678  ↻ (source updated)
    └─ tmux  @ ghi9012

$ skillforge sync --diff
  ↻ agents-grimoire has remote changes: abc1234 -> def5678
```

## 3. Command Mapping

| Feature | Command |
|---------|---------|
| Diagnostics | `skillforge doctor` |
| Dirty state | Auto in `skill list`, `sync --diff` |

## 4. Data Model Changes

### Grimoire (additions)
```toml
version = 1
source = "https://github.com/user/skills"
commit = "abc123"
tree_hash = "sha256:..."
installed_at = 2026-04-28T...
```

## 5. Out of Scope (YAGNI)

- Interactive TUI
- Suggest command
- Plugin system
- Auto-update mechanism
- Cloud sync
- Team collaboration

---

## Appendix: v1→v2 Migration

When implementing v2, ensure:
- [ ] Backward compatibility with v1 grimoire (no tree_hash = skip check)
- [ ] Graceful handling of missing git (doctor still runs other checks)
- [ ] Incremental tree_hash computation for large skills
