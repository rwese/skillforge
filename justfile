# Build the skillforge binary
build:
    go build -o skillforge ./cmd/skillforge/

# Install skillforge to ~/.local/bin
install: build
    mkdir -p ~/.local/bin
    mv skillforge ~/.local/bin/skillforge
    chmod +x ~/.local/bin/skillforge

# Run skillforge with optional args
run *args:
    go run ./cmd/skillforge/ {{args}}

# Run tests
test:
    go test ./...

# Clean build artifacts
clean:
    rm -f skillforge

# Build with ldflags for release
release:
    go build -ldflags="-s -w" -o skillforge ./cmd/skillforge/
