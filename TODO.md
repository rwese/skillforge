## Goal

Allow selective agent activation during `skillforge setup detect` instead of auto-configuring all detected agents.

## Implementation Plan

### Core Logic
- [x] Create interactive checkbox selector for agents (`internal/agents/select.go`)
- [x] Modify `runSetupDetect` to use selector instead of confirm-all
- [x] Handle already-configured agents (pre-selected, toggleable)
- [x] Add summary review step (skip with `--yes`)

### Polish
- [x] Update lipgloss styling to match existing color scheme
- [x] Handle zero-selection exit gracefully
- [ ] Test with existing config (pre-selected behavior)

### Verification
- [x] `go build` succeeds
- [x] `go test ./...` passes
- [ ] Manual test: `skillforge setup detect` with interactions

## Completed

- [x] Commit: `579f57b feat(setup): add interactive agent selection to setup detect`
