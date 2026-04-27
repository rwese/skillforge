# skillforge-ng

A focused CLI for managing agent skills from git repositories.

## Features

- **Cache repositories** - Clone and keep skill repositories up-to-date
- **Install skills** - Copy skills to agent skill directories with version tracking
- **Search** - Find skills across cached repositories
- **Multiple targets** - Support skills for different agents (pi, claude, etc.)
- **Auto-detect scope** - Uses local config in git repos, falls back to global

## Installation

```bash
go install
```

Or build from source:

```bash
git clone https://github.com/rwese/skillforge-ng
cd skillforge-ng
go build -o skillforge ./cmd/skillforge/
```

## Quick Start

```bash
# Add a skill repository
skillforge repo add https://github.com/user/skills

# List available skills
skillforge skill search "docker"

# Install a skill
skillforge skill install docker

# List installed skills
skillforge skill list

# Update skills
skillforge skill update --check  # Check for updates
skillforge skill update         # Apply updates
```

## Commands

### Repository Management

```bash
skillforge repo add <url> [-b branch]   # Add repository
skillforge repo list [-f json]           # List cached repos
skillforge repo remove <name>            # Remove repository
skillforge repo update [-c] [name]       # Update (use -c to check only)
```

### Skill Management

```bash
skillforge skill install <name> [-t target]  # Install skill
skillforge skill list [-t target] [-f json]  # List installed skills
skillforge skill remove <name> [-t target]   # Remove skill
skillforge skill search <query> [-f json]     # Search skills
skillforge skill update [-c] [-t target]     # Update installed skills
```

### Target Management

```bash
skillforge target list [-f json]        # List targets
skillforge target add <name> <path> [-e] # Add target (use -e to enable)
skillforge target remove <name>          # Remove target
skillforge target enable <name>          # Enable target
skillforge target disable <name>        # Disable target
```

## Configuration

Config files (TOML format):

- **Global**: `~/.config/skillforge/config.toml`
- **Local**: `.skillforge/config.toml` (in git repo or cwd)

Example config:

```toml
[cache]
path = "~/.cache/skillforge/repos"

[targets.pi]
path = "~/.pi/agent/skills/"
enabled = true
```

### Scope

By default, skillforge auto-detects config scope:

- Use `--local` or `--global` to force a specific scope
- Local config in git repos takes precedence

## Output Formats

All list commands support `-f json` for JSON output:

```bash
skillforge target list -f json
skillforge repo list -f json
skillforge skill list -f json
```

## Shell Completion

```bash
# Bash
skillforge completion bash >> ~/.bashrc

# Zsh
skillforge completion zsh > "${fpath[1]}/_skillforge"

# Fish
skillforge completion fish > ~/.config/fish/completions/skillforge.fish
```

## License

MIT
