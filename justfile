# Build the skillforge binary
build:
    go build -o skillforge ./cmd/skillforge/

# Install skillforge to $HOME/go/bin/
install: build
    mkdir -p $HOME/go/bin
    mv skillforge $HOME/go/bin/skillforge
    chmod +x $HOME/go/bin/skillforge

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
