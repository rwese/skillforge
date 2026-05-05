#!/bin/bash
set -e

REPO="rwese/skillforge"
BINARY="skillforge"

# Determine OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux*)  OS="linux" ;;
    Darwin*) OS="darwin" ;;
    *)       echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)       echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

DEST="${DEST:-$HOME/.local/bin}"
mkdir -p "$DEST"

# Get latest release version
VERSION=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep -o '"tag_name": ".*"' | cut -d'"' -f4)
VERSION="${VERSION#v}"

if [ -z "$VERSION" ]; then
    echo "No release found, building from source..."
    TMPDIR=$(mktemp -d)
    git clone --depth 1 https://github.com/$REPO "$TMPDIR/skillforge"
    cd "$TMPDIR/skillforge"
    go build -ldflags="-s -w" -o "${DEST}/${BINARY}" ./cmd/skillforge/
    chmod +x "${DEST}/${BINARY}"
    rm -rf "$TMPDIR"
    echo "Installed to ${DEST}/${BINARY}"
    echo "Add ${DEST} to your PATH if needed"
    exit 0
fi

# Try goreleaser archive format first (skillforge_vX.X.X_OS_ARCH.tar.gz)
ARCHIVE_URL="https://github.com/$REPO/releases/download/v${VERSION}/${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"

echo "Installing skillforge v${VERSION} for ${OS}/${ARCH}..."
echo "Downloading from $ARCHIVE_URL"

TMPDIR=$(mktemp -d)
if curl -fsSL "$ARCHIVE_URL" -o "$TMPDIR/archive.tar.gz"; then
    tar xzf "$TMPDIR/archive.tar.gz" -C "$DEST"
    chmod +x "${DEST}/${BINARY}"
    rm -rf "$TMPDIR"
    echo "Installed to ${DEST}/${BINARY}"
    echo "Add ${DEST} to your PATH if needed"
else
    rm -rf "$TMPDIR"
    echo "No pre-built binary found for ${OS}/${ARCH}, building from source..."
    TMPDIR=$(mktemp -d)
    git clone --depth 1 https://github.com/$REPO "$TMPDIR/skillforge"
    cd "$TMPDIR/skillforge"
    go build -ldflags="-s -w" -o "${DEST}/${BINARY}" ./cmd/skillforge/
    chmod +x "${DEST}/${BINARY}"
    rm -rf "$TMPDIR"
    echo "Installed to ${DEST}/${BINARY}"
    echo "Add ${DEST} to your PATH if needed"
fi
