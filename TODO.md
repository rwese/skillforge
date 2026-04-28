# skillforge-ng TODO

## v1.2 - Usability Improvements ✓ (completed)
See: [Usability Improvements PRD](./docs/prds/skillforge-usability-improvements/README.md)

### Phase 1: Config Merge Fix ✓
- [x] Write failing unit tests for config merging
- [x] Fix `config.Load()` to merge global + local
- [x] Fix `config.Save()` to preserve all scopes
- [x] Verify all tests pass

### Phase 2: Onboarding Hints ✓
- [x] Create hint formatting function (hints.go)
- [x] Add hints to `skill install` errors
- [x] Add hints to `skill search` errors
- [x] Add hints to `repo` command errors
- [x] Add hints to `target` command errors

### Phase 3: Output Improvements ✓
- [x] Add lipgloss dependency
- [x] Create `color.go` with color definitions
- [x] Create `output.go` with formatting functions
- [x] Implement color auto-detection
- [x] Update `skill list` with table/compact/json
- [x] Update `target list` with table/compact/json
- [x] Update `repo list` with table/compact/json
- [x] Implement auto-compact for pipes

### Phase 4: Integration Tests ✓
- [x] Set up integration test environment (temp cwd + local config)
- [x] Test full `target add/list/remove` cycle
- [x] Test scope merging with temp directories
- [x] Test output formats (table/json/compact)
- [x] Test hints in error messages
- [x] Test NO_COLOR environment variable

---

## v1.1 - Completed ✓

See: [Stability & Quality PRD](./docs/prds/skillforge-ng-stability-quality/README.md)

### Bug Fixes ✓
- [x] Fix cache.go Clone branch logic
- [x] Add ensureTargetDir call in skill.go
- [x] Remove unused commit variable
- [x] Fix spinner goroutine leak

### Testing ✓
- [x] Add config_test.go (82.7% coverage)
- [x] Add search_test.go (100% coverage)
- [x] Add grimoire_test.go (types tested)
- [x] Add repo_test.go integration tests (64.4% coverage)
- [x] Verify `go test ./...` passes
- [x] Verify `go vet ./...` is clean

### UX Polish ✓
- [x] Add `--dry-run` flag (preview only)
- [x] Add `-y/--yes` flag (skip confirmations)
- [x] Add `-v/--verbose` flag (debug output)
- [x] Add confirmation prompts for destructive actions
- [x] Improve error messages with context
- [x] Add progress for file copy operations

### Documentation ✓
- [x] Refresh README.md with troubleshooting
- [x] Create CONTRIBUTING.md
- [x] Create CHANGELOG.md

---

## v2 Future
See: [v2 PRD](./docs/prds/skillforge-ng-v2/README.md)

- [ ] `skillforge doctor` command (config, cache, skill health checks)
- [ ] Dirty state detection (tree_hash in grimoire)
- [ ] Auto-check on `skill list` and `skill update`

---

## Project Resources

| Resource | Location |
|----------|----------|
| Code Review | `docs/reviews/2026-04-28-code-review.md` |
| Improvement Plan | `docs/plans/improvement-plan.md` |
| v1 PRD | `docs/prds/skillforge-ng-v1/README.md` |
| v2 PRD | `docs/prds/skillforge-ng-v2/README.md` |
| v1.1 PRD | `docs/prds/skillforge-ng-stability-quality/README.md` |
| Documentation PRD | `docs/prds/skillforge-ng-documentation/README.md` |
| Usability PRD | `docs/prds/skillforge-usability-improvements/README.md` |
