---
name: skillforge
description: Manage agent skills with the SkillForge CLI. Use when installing, configuring, searching for, loading, or removing agent skills from cached git repositories across multiple agents (pi, codex, claude). Covers setup, repo caching, skill discovery, install, search, load, sync, and target configuration.
---

# SkillForge

`skillforge` is a CLI that clones skill repositories to a local cache
and symlinks discovered skills into agent skill directories. One cache,
many agents, symlinks only.

## Quick Reference

| Goal | Command |
|------|---------|
| Install skillforge | `brew install rwese/skillforge/skillforge` (or `go install github.com/rwese/skillforge/cmd/skillforge@latest`) |
| Configure agents on first run | `skillforge setup` |
| Add a skill repo | `skillforge repo add <url> [-b branch] [--alias name]` |
| Search skills | `skillforge skill search <query>` |
| **Load a skill into a temp dir** | `skillforge skill load <name>` |
| Install a skill (symlink to agents) | `skillforge skill install <name> [-t target]` |
| List / remove a skill | `skillforge skill list` / `skillforge skill remove <name>` |
| Refresh cache + re-link | `skillforge sync --fix-all` |
| Re-link broken symlinks only | `skillforge sync --fix-broken-symlinks` |

## Install

Pick one method:

```bash
brew install rwese/skillforge/skillforge
```

```bash
go install github.com/rwese/skillforge/cmd/skillforge@latest
```

```bash
git clone https://github.com/rwese/skillforge
cd skillforge && ./install.sh
```

Verify: `skillforge --version`.

## First-time setup

```bash
skillforge setup                # interactive wizard, detects pi/codex/claude
```

Then add a skill repo and search it:

```bash
skillforge repo add https://github.com/rwese/agents-grimoire
skillforge skill search docker
```

## Search

```bash
skillforge skill search docker
skillforge skill search "docker container"
skillforge skill search postgres database --format json
```

Multi-word queries are space-joined. Nested skill names
(`architecture/event-sourced-commands`) match on each path segment,
so `skill search architecture` finds them.

## Load a skill (no install)

`skill load` copies a cached skill into a fresh temp directory and
prints its `SKILL.md` to stdout. Use it when you want to inspect or
hand off a skill without linking it into an agent's skills directory.

```bash
skillforge skill load docker
# Skill loaded at: /tmp/skillforge-049-3db/docker
#
# SKILL.md:
#
# # docker
# ...
```

### Output format

The temp dir is laid out as:

```
/tmp/skillforge-<hash>/<skill-name>/
```

where `<hash>` is the first three bytes of the SHA256 of the current
working directory, formatted as `xxx-xxx` for readability. Nested
skill names keep their category path on disk, so
`skillforge skill load architecture/event-sourced-commands` lands at:

```
/tmp/skillforge-<hash>/architecture/event-sourced-commands/
```

### Cleanup

The directory is **left in place** — the caller removes it when done.
The hash segment changes per cwd, so concurrent loads from different
working directories never collide.

## Global flags

| Flag | Effect |
|------|--------|
| `-s, --scope {global,local,all}` | Config scope. Default: local. |
| `-n, --dry-run` | Preview without applying. |
| `-y, --yes` | Skip confirmations. |
| `-v, --verbose` | Debug output. |
| `--config <path>` | Override config file path. |

Config lives at `~/.config/skillforge/config.toml` (global) and
`./.skillforge.toml` (local / per-project).

## Common workflows

```bash
# Preview what sync would change without applying
skillforge sync

# Apply repo updates + link missing skills + repair broken symlinks
skillforge sync --fix-all

# Add a skill repo, search it, install one skill
skillforge repo add https://github.com/user/agent-skills --alias skills
skillforge skill search git
skillforge skill install my-git-fu

# Install only to a specific target
skillforge skill install my-git-fu -t pi
```