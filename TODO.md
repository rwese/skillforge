# Target Model Migration: GlobalPath/LocalPath

## Overview
Migrate targets from single `Path` to dual `GlobalPath`/`LocalPath` model.

## Tasks

- [x] 1. Update data model: rename `Path` → `LocalPath`, add `GlobalPath`
- [x] 2. Update `target add` command: positional args `name globalPath localPath`
- [x] 3. Update `target list` command: display both paths
- [x] 4. Update `skill install/remove/list`: use correct path based on scope
- [x] 5. Migrate global config: add `GlobalPath` to `pi`, remove `pi-local`
- [x] 6. Add regression tests for new model
- [x] 7. Update config TOML examples
- [x] 8. Build and manual verification
- [x] 9. Commit changes

## Acceptance Criteria

- [x] `target add pi ~/.pi/agent/skills .pi/skills` creates target with both paths
- [x] `target list` shows globalPath and localPath columns
- [x] `skill install tmux` installs to localPath from local config
- [x] `skill install tmux -s global` installs to globalPath from global config
- [x] Global config has `pi` with both paths set
- [x] `pi-local` target removed from all configs
- [x] All tests pass
