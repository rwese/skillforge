## Goal

Allow selective agent activation during `skillforge setup detect` instead of auto-configuring all detected agents.

## Status: Complete ✅

### Verified via tmux Testing

| Test | Status | Notes |
|------|--------|-------|
| `go build` | ✅ | |
| `go test ./...` | ✅ | All 6 packages pass |
| TUI renders | ✅ | Shows agent list with checkboxes |
| Arrow keys | ✅ | Move cursor up/down |
| Space | ✅ | Toggle selected item |
| `n` key | ✅ | Deselect all |
| `s` key | ✅ | Select all |
| Enter | ✅ | Confirm selection |
| Summary prompt | ✅ | Shows selected agents |
| `y` confirm | ✅ | Saves to config |
| `--yes` flag | ✅ | Skips interactive mode |

## Commits

| Commit | Description |
|--------|-------------|
| `579f57b` | feat(setup): add interactive agent selection |
| `4afa957` | test(agents): add unit tests |
| `de47aab` | fix(setup): skip interactive with --yes |
| `472d22f` | docs: update TODO |
| `43ad8b8` | fix(agents): handle \r and \n for Enter |
| `be4e200` | docs: mark feature complete |
| `8103fd2` | fix(tui): use raw terminal mode for tmux |

## Usage

```bash
# Interactive mode
skillforge setup detect

# Non-interactive
skillforge setup detect --yes
```

### Keyboard Controls

| Key | Action |
|-----|--------|
| ↑/↓ | Navigate |
| Space | Toggle |
| a | Toggle all configured |
| s | Select all |
| n | Deselect all |
| Enter | Confirm |
| q | Quit |
