# PRD: SkillForge-NG v2 - Future Enhancements

**Version:** 1.0.0-draft **Date:** 2026-04-27 **Status:** Future **Target:** v2.0

---

## 1. Executive Summary

Features deferred from v1 that require more design work or additional dependencies.

## 2. Planned Features

### 2.1 Interactive TUI

**Description:** A full-screen terminal UI for visual skill management.

**Commands affected:** `skillforge manage` (new)

**UI Components:**
- Repository browser with skill previews
- Target management panel
- Install/update workflow with confirmation
- Progress visualization for git operations

**Rationale:** Complex to implement well; CLI covers 90% of use cases.

### 2.2 Suggest Command

**Description:** Analyze project context and suggest relevant skills.

**Input:**
- Project files (via git or explicit paths)
- Language detection
- Framework detection

**Output:**
- Ranked skill recommendations
- Match confidence scores
- One-click install

**Rationale:** Requires ML/heuristics; may need external service.

### 2.3 Dirty State Detection

**Description:** Detect when installed skills differ from their source.

**Scenarios:**
- Source repo updated but skill not re-installed
- Skill manually modified after installation
- Target directory deleted

**Implementation:**
- Compare installed tree_hash vs source
- Track modifications in grimoire
- `skillforge doctor` command for diagnostics

**Rationale:** Requires tree_hash computation (skipped in v1 for simplicity).

## 3. Technical Considerations

| Feature | Complexity | Dependencies |
|---------|------------|--------------|
| TUI | High | bubble, lipgloss, or similar |
| Suggest | Medium | Pattern matching, optional ML |
| Dirty State | Low | Already have commit tracking |

## 4. Command Mapping

| Feature | New Command |
|---------|-------------|
| TUI | `skillforge manage` |
| Suggest | `skillforge suggest [path]` |
| Diagnostics | `skillforge doctor` |

## 5. Out of Scope

- Plugin system
- Auto-update mechanism
- Cloud sync
- Team collaboration features

---

## Appendix: v1→v2 Migration Checklist

When implementing v2, ensure:
- [ ] Backward compatibility with v1 config format
- [ ] Graceful degradation when TUI unavailable (TERM=dumb)
- [ ] Migration path for grimoire files without tree_hash
- [ ] Deprecation warnings for v1-only features
