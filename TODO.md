## Scoped Targets Implementation

Goal: Implement agent-centric target scoping with global/local paths and a setup wizard.

### Tasks

#### Phase 1: Agent Config Types
- [x] Create `internal/agents/agents.go`
  - [x] `Agent` struct with `Global`, `Local` paths
  - [x] `AgentsConfig` struct
  - [x] `LoadAgents(scope Scope)` function
  - [x] `MergeAgents(global, local)` function
  - [x] `KnownAgents` map (pi, codex, claude defaults)

#### Phase 2: Setup Wizard
- [x] Create `cmd/skillforge/setup.go`
  - [x] `setup` command with `detect` subcommand
  - [x] `detectKnownAgents()` — scan filesystem for known agents
  - [x] Interactive confirmation flow
  - [x] Write to `~/.config/skillforge/agents.toml`

#### Phase 3: Update Skill Operations
- [x] Update `skill list` to use agents config
- [x] Update `skill install` to use agents config
- [x] Update `skill update` to use agents config
- [x] Update `skill remove` to use agents config

#### Phase 4: Tests
- [x] Add tests for `internal/agents/`
- [ ] Add integration tests for setup wizard

### In Progress

-

### Blocked

-

### Done

- [x] Design finalized
- [x] Phase 1: Agent config types (internal/agents/)
- [x] Phase 2: Setup wizard (cmd/skillforge/setup.go)
- [x] Phase 3: Skill operations use agents config
- [x] Phase 4: Tests for internal/agents/
