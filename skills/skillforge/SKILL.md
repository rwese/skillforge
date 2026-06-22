---
name: skillforge
description: search skills and load skills for various tasks.
---

# SkillForge

`skillforge` is a CLI that clones skill repositories to a local cache
and symlinks discovered skills into agent skill directories. One cache,
many agents, symlinks only.

## Quick Reference

| Goal | Command |
|------|---------|
| Install skillforge | `go install github.com/rwese/skillforge/cmd/skillforge@latest` |
| Configure agents on first run | `skillforge setup` |
| Add a skill repo | `skillforge repo add <url> [-b branch] [--alias name]` |
| Search skills | `skillforge skill search <query>` |
| **Load a skill into a temp dir** | `skillforge skill load <name>` |

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

Then add a skill repo and search it:

```bash
skillforge repo add https://github.com/rwese/agents-grimoire
skillforge skill search docker
```

## Search

```bash
skillforge skill search docker
skillforge skill search "docker container"
```

## Load a skill

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
