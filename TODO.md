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
- [x] Fix --yes flag to skip interactive selector entirely

### Verification
- [x] `go build` succeeds
- [x] `go test ./...` passes
- [x] Manual test: `--yes` flag works (verified)
- [x] Manual test: UI renders correctly (tmux capture verified)
- [x] Manual test: 'n' key deselects all (verified)
- [ ] Manual test: Arrow keys and Space toggle (requires terminal)

## Completed

- [x] Commit: `579f57b feat(setup): add interactive agent selection to setup detect`
- [x] Commit: `4afa957 test(agents): add unit tests for selectable agent logic`
- [x] Commit: `de47aab fix(setup): skip interactive selector with --yes flag`

## Notes

- Interactive testing via tmux stdin has limitations for arrow keys
- Character keys ('n', 's', 'a', space, enter) work through tmux
- Full interactive testing requires manual terminal session
