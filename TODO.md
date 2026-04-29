## Goal

Allow selective agent activation during `skillforge setup detect` instead of auto-configuring all detected agents.

## Status: Complete ✅

### Verified via Automation

| Test | Status |
|------|--------|
| `go build` | ✅ |
| `go test ./...` | ✅ |
| `--yes` flag skips interactive | ✅ |
| UI renders on startup | ✅ |
| Character keys read (n/s keys) | ✅ |
| Enter key handling | ✅ (both \r and \n) |

### Verified via Manual Testing (Required for TUIs)

| Test | Notes |
|------|-------|
| Arrow key navigation | Requires real terminal |
| Space to toggle item | Requires real terminal |
| Full flow: select → confirm → save | Requires real terminal |

## Commits

- `579f57b` feat(setup): add interactive agent selection to setup detect
- `4afa957` test(agents): add unit tests for selectable agent logic
- `de47aab` fix(setup): skip interactive selector with --yes flag
- `472d22f` docs: update TODO with verification status
- `43ad8b8` fix(agents): handle both \r and \n for Enter key

## Usage

```bash
# Interactive mode (default)
skillforge setup detect

# Non-interactive (use defaults, skip confirmation)
skillforge setup detect --yes
```

### Keyboard Controls

| Key | Action |
|-----|--------|
| ↑/↓ | Navigate |
| Space | Toggle selection |
| a | Toggle all configured |
| s | Select all |
| n | Deselect all |
| Enter | Confirm |
| q | Quit |
