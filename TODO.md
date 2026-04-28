# Skill Sync Feature Implementation

## Tasks

### Phase 1: Verification First - Add flags to existing sync command
- [x] Review existing sync.go structure
- [x] Add `--agent`, `--scope`, `--skip-agent-sync` flags to sync command
- [x] Verify flags register correctly

### Phase 2: Core Logic - Implement agent sync detection
- [x] Collect all installed skills per agent+scope
- [x] Build union set of all skill names
- [x] Identify missing skills per agent+scope
- [x] Print missing skills in dry-run mode

### Phase 3: Apply Logic - Install missing skills
- [x] Use existing `InstallSkill` infrastructure from skill.go
- [x] Track install results
- [x] Error handling for failed installs

### Phase 4: Polish - Edge cases & tests
- [x] Empty agent handling (gracefully handles empty/no agents)
- [x] Verbose output improvements (shows counts, targets)
- [x] Unit tests for sync logic
- [ ] Update documentation

## Acceptance Criteria
- [x] `skillforge sync` syncs repos + updates skills + syncs across agents
- [x] `skillforge sync --check` shows all three operations preview
- [x] Scope flag (`--scope global|local|auto`) respects agent config scopes
- [x] Agent flag (`--agent <name>`) syncs only specific agent
- [x] No cross-scope contamination (global stays global, local stays local)
- [x] Handles agents with no skills gracefully
- [x] Verbose output shows what's happening
- [x] Tests pass
