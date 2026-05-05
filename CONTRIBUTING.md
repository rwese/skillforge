# Contributing to skillforge

Thank you for your interest in contributing!

## Development Setup

```bash
# Clone the repository
git clone https://github.com/rwese/skillforge
cd skillforge

# Install dependencies
go mod download

# Build
go build -o skillforge ./cmd/skillforge/

# Run tests
go test ./...

# Run with verbose output
./skillforge -v [command]
```

## Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Run `go vet` to check for issues
- Aim for clear, simple code

## Testing

All new features should include tests:

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Run specific package tests
go test ./internal/config -v
```

## Commit Messages

Use Conventional Commits format:

```
feat: add docker skill installer
fix: resolve cache clone issue
docs: update README
test: add config loading tests
```

## Project Structure

```
skillforge/
├── cmd/skillforge/     # CLI commands (cobra)
├── internal/
│   ├── config/        # Configuration loading
│   ├── repo/          # Repository operations
│   └── search/        # Skill search
├── pkg/grimoire/       # Skill metadata types
└── docs/              # Documentation
```

## Pull Requests

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make changes with tests
4. Ensure tests pass: `go test ./...`
5. Commit with clear message
6. Push and create PR

## Reporting Issues

Include:
- Command executed
- Expected behavior
- Actual behavior
- Go version
- OS

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
