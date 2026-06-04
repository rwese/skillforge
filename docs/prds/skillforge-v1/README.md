# PRD: SkillForge-NG

## Project Name: skillforge

**Version:** 1.0.0-draft **Date:** 2026-04-27 **Status:** Planning **Target:** MVP CLI

---

## 1. Executive Summary

**Problem Statement**

Current skillforge has accumulated significant complexity:
- 6700+ lines across 43 Go files
- 16+ commands with overlapping functionality
- Complex multi-scope handling (global/local with inheritance)
- Heavy TUI for basic operations
- Over-engineered search (inverted index, weighted scoring)
- Dirty state detection across scopes
- Suggest feature tightly coupled

**Proposed Solution**

Trim to core essence: a focused CLI for managing agent skills from git repositories.

**Core capabilities:**
1. Cache skill repositories (git clone/fetch)
2. Install skills to agent directories with version tracking
3. Search and list skills
4. Manual update mechanism

---

## 2. Goals & Non-Goals

**Goals**

1. Simplify CLI interface with structured subcommands
2. Retain full grimoire metadata (tree_hash, commit, source)
3. Support multiple agents via extensible target system
4. Auto-detect config scope (local if `.skillforge/` in cwd/git-root, else global)
5. Provide essential operations: add repo, install, remove, list, search, update
6. Single executable, no external dependencies beyond git

**Non-Goals**

1. Interactive TUI (defer to v2)
2. Suggest command (defer to v2)
3. Dirty state detection across scopes (defer)
4. Auto-update (manual only)
5. Plugin system (defer)
6. Complex inheritance model

---

## 3. Technical Architecture

### Technology Stack

| Layer | Technology | Rationale |
|-------|------------|-----------|
| Language | Go 1.21+ | Required by user preference |
| CLI | cobra + bubble/tease | Lightweight cobra setup |
| Config | TOML | User preference |
| Git ops | exec git | Reliable, no external libs |
| Search | Simple keyword match | Avoid over-engineering |

### Project Structure

```
skillforge/
├── cmd/
│   └── skillforge/
│       ├── main.go           # Entry point
│       ├── root.go           # Root command + flags
│       ├── repo.go           # repo subcommands
│       ├── skill.go          # skill subcommands
│       └── target.go         # target subcommands
├── internal/
│   ├── config/
│   │   ├── config.go         # Config loading/writing
│   │   └── scope.go         # Scope detection (cwd/git-root)
│   ├── repo/
│   │   ├── cache.go         # Git clone/fetch/pull
│   │   ├── grimoire.go      # .grimoire read/write
│   │   └── discover.go      # Find skills in repos
│   ├── skill/
│   │   ├── install.go       # Copy skill to targets
│   │   ├── remove.go        # Remove skill from targets
│   │   └── list.go          # List installed skills
│   └── search/
│       └── search.go        # Simple keyword search
├── pkg/
│   └── grimoire/
│       └── types.go         # Grimoire struct
└── docs/
    └── prds/
        └── skillforge-v1/
            └── README.md
```

### Configuration

**Config file locations:**
- Global: `~/.config/skillforge/config.toml`
- Local: `<git-root>/.skillforge/config.toml` (if in git repo) or `<cwd>/.skillforge/config.toml`

**Config format:**
```toml
[cache]
path = "~/.cache/skillforge/repos"

[targets.pi]
globalPath = "~/.pi/agent/skills/"
localPath = ".pi/skills"
enabled = true
```

**Grimoire format (`.grimoire` in skill directory):**
```toml
version = 1
source = "https://github.com/user/skills-repo"
commit = "abc123..."
installed_at = "2026-04-27T10:00:00Z"
```

---

## 4. Interface Specification

### Main Entry

```bash
skillforge [OPTIONS] <COMMAND>

OPTIONS:
  -g, --global    Use global config only
  -l, --local     Use local config only
  -h, --help      Help
```

### Scope Detection Logic

1. If `--global` flag: use global config only
2. If `--local` flag: use local config only
3. Else auto-detect:
   - Search for `.skillforge/config.toml` starting from cwd
   - If found in git working tree, stop at git root
   - If not in git, use cwd
   - If no local config found, fall back to global

### Subcommands

#### Repository Management

```bash
skillforge repo add <url> [OPTIONS]

OPTIONS:
  -b, --branch <name>   Branch to track (default: main)

EXAMPLES:
  skillforge repo add https://github.com/user/skills
  skillforge repo add https://github.com/org/grimoire --branch dev
```

```bash
skillforge repo list [OPTIONS]

OPTIONS:
  -f, --format <format>   Output: text, json (default: text)
```

```bash
skillforge repo remove <url>
```

```bash
skillforge repo update [OPTIONS]

OPTIONS:
  -c, --check    Check for updates without applying
```

#### Skill Management

```bash
skillforge skill install <name> [OPTIONS]

OPTIONS:
  -t, --target <name>   Target agent (default: all enabled)

EXAMPLES:
  skillforge skill install github
  skillforge skill install docker --target pi
```

```bash
skillforge skill list [OPTIONS]

OPTIONS:
  -t, --target <name>   Filter by target
  -f, --format <format>   Output: text, json (default: text)
```

```bash
skillforge skill remove <name> [OPTIONS]

OPTIONS:
  -t, --target <name>   Target agent (default: all)
```

```bash
skillforge skill search <query>
```

```bash
skillforge sync [OPTIONS]

OPTIONS:
  --diff                  Show incoming repository changes
  --fix-sync-repos        Update cached repositories
  --fix-outofsync-agents  Link missing agent skills
  --fix-all               Apply repository and agent fixes
```

#### Target Management

```bash
skillforge target list [OPTIONS]

OPTIONS:
  -f, --format <format>   Output: text, json (default: text)
```

```bash
skillforge target add <name> <globalPath> <localPath> [OPTIONS]

OPTIONS:
  -e, --enable    Enable target after creation
```

```bash
skillforge target remove <name>
```

```bash
skillforge target enable <name>
skillforge target disable <name>
```

### Output Examples

```bash
$ skillforge repo list
Cached Repositories:
  ✓ github.com/user/skills     (15 skills, updated 2h ago)
  ✓ github.com/org/grimoire    (8 skills, updated 1d ago)
```

```bash
$ skillforge skill list
Installed Skills:
  github.com/user/skills/
    ├─ github        @ abc1234  (pi)
    ├─ docker        @ def5678  (pi)
    └─ keyboard      @ ghi9012  (pi)
```

```bash
$ skillforge skill search "git"
Search results for "git":
  1. github       Use GitHub APIs          github.com/user/skills
  2. git-ops     Git workflow automation  github.com/org/grimoire
```

---

## 5. Feature Roadmap

### Phase 1: Core MVP

- [ ] Project scaffold with cobra
- [ ] Config loading/writing (TOML)
- [ ] Scope detection (cwd/git-root)
- [ ] Target management (add, list, remove, enable, disable)
- [ ] Repo management (add, list, remove, update)
- [ ] Skill installation with grimoire
- [ ] Skill listing
- [ ] Skill removal
- [ ] Simple search
- [ ] Skill update (check + apply)

### Phase 2: Polish

- [ ] Tree hash computation for change detection
- [ ] Better error messages
- [ ] Progress indicators for long operations
- [ ] JSON output for all commands
- [ ] Completions (bash, zsh, fish)

### Phase 3: Future (Deferred)

- [ ] Interactive TUI
- [ ] Suggest command (project analysis)
- [ ] Dirty state detection
- [ ] Plugin system
- [ ] Auto-update feature

---

## 6. Decisions

| Question | Decision | Rationale |
|----------|----------|-----------|
| Aliases | No | Simplicity, avoid ambiguity |
| Conflicts | Error | Clear behavior, force explicit |
| tree_hash | Skip | Simplifies grimoire, sufficient with commit |
| Cache retention | Manual only | User controls when to clean |

---

## 7. Success Criteria

- [ ] Single binary, no external runtime deps
- [ ] All operations work with both global and local configs
- [ ] Grimoire metadata tracks source, commit, installed_at
- [ ] `sync --diff` correctly identifies incoming repository changes
- [ ] Search returns relevant results within 100ms for 100 skills
- [ ] Install/remove operations are atomic (fail-safe)
- [ ] Tests cover core operations (>80% coverage target)
- [ ] Clean separation: config, repo, skill packages

---

## 8. Repository Location

```
/Users/wese/Repos/github.com/rwese/skillforge/
```

---

## Appendix: Migration from skillforge

| Aspect | Current | New |
|--------|---------|-----|
| Commands | 16+ flat | 3 groups (repo, skill, target) |
| Scope | Complex inheritance | Auto-detect + explicit flags |
| TUI | Full interactive | CLI only (defer) |
| Search | Inverted index | Simple keyword |
| Dirty state | Full detection | Deferred |
| Symlinks | Copy (not symlink) | Copy (same) |
| Config | TOML | TOML (same) |

### Command Mapping

| Old Command | New Command |
|-------------|-------------|
| `skillforge add` | `skillforge repo add` |
| `skillforge install` | `skillforge skill install` |
| `skillforge remove` | `skillforge skill remove` |
| `skillforge list` | `skillforge skill list` |
| `skillforge search` | `skillforge skill search` |
| `skillforge update` | `skillforge sync --fix-all` |
| `skillforge update --check` | `skillforge sync --diff` |
| `skillforge doctor` | (defer to v2) |
| `skillforge manage` | (defer to TUI) |
| `skillforge suggest` | (defer) |
| `skillforge sync` | `skillforge repo update` |
| `skillforge config` | `skillforge target *` |
| `skillforge repo *` | (new structured repo commands) |
