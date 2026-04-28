## Scoped Targets Implementation

Goal: Implement agent-centric target scoping with global/local paths and a setup wizard.

### Done

- [x] Design finalized
- [x] Phase 1: Agent config types (internal/agents/)
- [x] Phase 2: Setup wizard (cmd/skillforge/setup.go)
- [x] Phase 3: Skill operations use agents config
- [x] Phase 4: Tests for internal/agents/

### Remaining

- [ ] Add integration tests for setup wizard
- [ ] Deprecate old targets system (or keep for backward compat)
- [ ] Update documentation

---

## Implementation Summary

### Files Created
- `internal/agents/agents.go` - Agent config types + loading
- `internal/agents/agents_test.go` - Tests
- `internal/agents/skills.go` - Skill resolution helpers
- `cmd/skillforge/setup.go` - Setup wizard

### Files Modified
- `cmd/skillforge/skill.go` - Use agents config instead of targets

### New Commands
```
skillforge setup detect   # Auto-detect known agents
skillforge setup list     # List configured agents
skillforge setup add      # Add agent manually
```

### Updated Commands
```
skillforge skill list --agent pi --scope global
skillforge skill install --agent pi --scope local <skill>
skillforge skill remove --agent pi <skill>
skillforge skill update --agent pi --scope local
```

### Config Location
`~/.config/skillforge/agents.toml`
