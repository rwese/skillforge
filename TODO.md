## Goal

Allow selective agent activation during `skillforge setup detect` instead of auto-configuring all detected agents.

## Implementation Plan

### Core Logic
- [x] Create interactive checkbox selector for agents (`internal/agents/select.go`)
- [x] Modify `runSetupDetect` to use selector instead of confirm-all
- [x] Handle already-configured agents (pre-selected, toggleable)
- [x] Add summary review step (skip with `--yes`)

### Polish
- [ ] Update lipgloss styling to match existing color scheme
- [ ] Handle zero-selection exit gracefully
- [ ] Test with existing config (pre-selected behavior)

### Verification
- [ ] `go build` succeeds
- [ ] `go test ./...` passes
- [ ] Manual test: `skillforge setup detect` with interactions
