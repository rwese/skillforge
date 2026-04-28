# skillforge-ng TODO

## v1.1 (Current - In Progress)
See: [Stability & Quality PRD](./docs/prds/skillforge-ng-stability-quality/README.md)

### Bug Fixes
- [x] Fix cache.go Clone branch logic
- [x] Add ensureTargetDir call in skill.go
- [x] Remove unused commit variable
- [x] Fix spinner goroutine leak

### Testing
- [x] Add config_test.go (82.7% coverage)
- [x] Add search_test.go (100% coverage)
- [x] Add grimoire_test.go (types tested)
- [x] Add repo_test.go integration tests (64.4% coverage)
- [x] Verify `go test ./...` passes
- [x] Verify `go vet ./...` is clean

### UX Polish
- [x] Add `--dry-run` flag (preview only)
- [x] Add `-y/--yes` flag (skip confirmations)
- [x] Add `-v/--verbose` flag (debug output)
- [x] Add confirmation prompts for destructive actions
- [x] Improve error messages with context
- [x] Add progress for file copy operations

### Documentation
See: [Documentation PRD](./docs/prds/skillforge-ng-documentation/README.md)

- [x] Refresh README.md with troubleshooting
- [x] Create CONTRIBUTING.md
- [x] Create CHANGELOG.md

---

## v2 Future
See: [v2 PRD](./docs/prds/skillforge-ng-v2/README.md)

- [ ] Interactive TUI (`skillforge manage`)
- [ ] `doctor` command for diagnostics
- [ ] `suggest` command for project analysis
- [ ] Dirty state detection with tree hashes
- [ ] `config validate` command
- [ ] Import/export functionality

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
