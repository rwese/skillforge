# skillforge-ng TODO

## Phase 1: Core MVP

### Project Setup
- [x] Initialize Go module
- [x] Create directory structure
- [x] Add cobra dependencies
- [x] Create root command with flags

### Config System
- [x] Config loading/writing (TOML)
- [x] Scope detection (cwd/git-root)
- [x] Default config generation

### Target Management
- [x] `target list` command
- [x] `target add` command
- [x] `target remove` command
- [x] `target enable` command
- [x] `target disable` command

### Repo Management
- [x] `repo add` command
- [x] `repo list` command
- [x] `repo remove` command
- [x] `repo update` command

### Skill Management
- [x] `skill install` command
- [x] `skill list` command
- [x] `skill remove` command
- [x] `skill search` command
- [x] `skill update` command (check + apply)

## Phase 2: Polish
- [x] JSON output for all list commands
- [ ] Progress indicators for long operations
- [x] Completions (bash, zsh, fish, powershell)

## Phase 3: Future (Deferred)
- [ ] Interactive TUI
- [ ] Suggest command
- [ ] Dirty state detection
