# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- docs: slim down v2 scope (YAGNI - removed TUI, suggest, plugin system)

## [v1.1] - 2026-04-28

### Added
- `--dry-run` flag for previewing operations
- `-y/--yes` flag for skipping confirmations
- `-v/--verbose` flag for debug output
- Confirmation prompts for destructive actions (remove commands)
- Progress reporting for file copy operations (with `--verbose`)
- Comprehensive test suite (config, search, grimoire, repo)

### Fixed
- Cache clone branch logic for non-main branches
- Spinner goroutine leak (added thread-safe stopping)
- Unused commit variable in repo add output
- Target directory creation before skill installation

### Changed
- Improved error messages with context

## [v1.0.0] - 2024-04-28

### Added
- Repository management (add, list, remove, update)
- Skill installation from cached repositories
- Skill search across repositories
- Target management for multiple agents
- Config scope detection (global/local/auto)
- JSON output format for all list commands
- Shell completion support
- Grimoire metadata for version tracking

### Features
- Cache git repositories with shallow clones
- Discover skills in `skills/` and `.agents/skills/` directories
- Auto-detect local config in git repositories
- Progress spinner for long operations
- Skill update checking and installation
