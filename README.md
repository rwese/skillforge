# skillforge-ng

A focused CLI for managing agent skills from git repositories.

## Features

- **Cache repositories** - Clone and keep skill repositories up-to-date
- **Install skills** - Copy skills to agent skill directories with version tracking
- **Search** - Find skills across cached repositories
- **Multiple targets** - Support skills for different agents (pi, claude, etc.)
- **Auto-detect scope** - Uses local config in git repos, falls back to global

## Installation

### Homebrew (macOS/Linux)

```bash
brew install rwese/skillforge-ng/skillforge
```

### Pre-built binary

```bash
# Clone and run install script
git clone https://github.com/rwese/skillforge-ng
cd skillforge-ng
./install.sh

# Or download a release manually from:
# https://github.com/rwese/skillforge-ng/releases
```

### Build from source

```bash
git clone https://github.com/rwese/skillforge-ng
cd skillforge-ng
go build -o skillforge ./cmd/skillforge/
# Place in PATH
mv skillforge ~/.local/bin/
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

- Use `-s global`, `-s local`, or `-s auto` to control config scope
- Local config in git repos takes precedence over global
- Help text shows detected scope when not auto

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

## Global Flags

```bash
-n, --dry-run          # Preview changes without applying
-y, --yes              # Skip confirmations
-v, --verbose          # Enable debug output
-s, --scope <scope>    # Config scope: global, local, or auto
```

## Troubleshooting

### No skills found

1. Check repositories: `skillforge repo list`
2. Add a repository: `skillforge repo add <url>`
3. Update repos: `skillforge repo update`

### Permission denied

Ensure the target directory exists and is writable:
```bash
mkdir -p ~/.pi/agent/skills/
```

### Config not found

Use `-s` to specify config scope:
```bash
skillforge -s local skill list
```

### Verbose debugging

Use `-v` flag for debug output:
```bash
skillforge -v skill install docker
```

## License

MIT
