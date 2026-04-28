# Feature: Skill Sync Across Agents

## Context

**Goal:** Extend existing `sync` command to also synchronize skills across agents within a defined scope. When a new agent is added, detect missing skills and offer to install them bidirectionally.

## Scope

**In Scope:**
- Extend existing `skillforge sync` command
- Bidirectional sync detection (what's missing where)
- Scope-aware: sync global↔global, local↔local (not cross-contaminating)
- Dry-run mode to preview changes
- Integration with existing `skill install` infrastructure

**Out of Scope:**
- Auto-sync on new agent detection (manual trigger only)
- Real-time monitoring / webhooks
- Conflict resolution (skip on conflict)

## Acceptance Criteria

- [ ] `skillforge sync` syncs repos + updates skills + syncs across agents
- [ ] `skillforge sync --check` shows all three operations preview
- [ ] Scope flag (`--scope global|local|auto`) respects agent config scopes
- [ ] Agent flag (`--agent <name>`) syncs only specific agent
- [ ] No cross-scope contamination (global stays global, local stays local)
- [ ] Handles agents with no skills gracefully
- [ ] Verbose output shows what's happening

## First Verifiable State

1. **[Verification]** `skillforge skill sync --dry-run` runs without error and shows missing skills
2. **[Core Logic]** `skillforge skill sync --apply` installs missing skills
3. **[Polish]** Scope filtering and agent filtering work correctly

## Implementation Notes

### Tech Decisions:
- Reuse `ListInstalledSkills` from `internal/repo`
- Reuse `InstallSkill` from `internal/repo`
- New command in `cmd/skillforge/sync.go`

### Key Files:
- `cmd/skillforge/sync.go` — extend existing command
- `cmd/skillforge/skill.go` — reference for flags/patterns, install logic
- `internal/agents/agents.go` — agent config
- `internal/repo/` — skill operations

### Algorithm:
```
1. Collect all installed skills per agent+scope
2. Build union set of all skills
3. For each agent+scope, find missing skills
4. Optionally install missing skills
```

## Incremental Plan

### 1. **[Verification First]** — Add flags to existing sync command
- Add `--agent`, `--scope`, `--skip-agent-sync` flags to existing sync.go
- Verify flags register correctly

### 2. **[Core Logic]** — Implement agent sync detection
- `ListInstalledSkills` for each agent+scope
- Compute union set of skill names
- Identify missing skills per agent+scope
- Print missing skills in dry-run mode

### 3. **[Apply Logic]** — Install missing skills
- Use existing `InstallSkill` infrastructure from skill.go
- Track install results
- Error handling for failed installs

### 4. **[Polish]** — Edge cases & tests
- Empty agent handling
- Verbose output improvements
- Unit tests for sync logic
- Update documentation

## Definition of Done

- [ ] `skillforge sync --check` shows agent sync preview
- [ ] `skillforge sync` syncs repos + skills + agents
- [ ] `skillforge sync --agent pi` syncs only pi
- [ ] `skillforge sync --scope global` syncs only global scope
- [ ] `skillforge sync --skip-agent-sync` skips agent synchronization
- [ ] No cross-scope contamination
- [ ] Tests pass
- [ ] Documentation updated
