#!/bin/sh
# MediaWiki MCP Server installer
# Usage: curl -fsSL https://raw.githubusercontent.com/olgasafonova/mediawiki-mcp-server/main/install.sh | sh

set -e

REPO="olgasafonova/mediawiki-mcp-server"
BINARY_NAME="mediawiki-mcp-server"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case "$OS" in
    darwin)
        PLATFORM="darwin-$ARCH"
        ;;
    linux)
        PLATFORM="linux-$ARCH"
        ;;
    mingw*|msys*|cygwin*)
        PLATFORM="windows-amd64"
        BINARY_NAME="mediawiki-mcp-server.exe"
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

# Get latest release
echo "Fetching latest release..."
LATEST=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
    echo "Failed to fetch latest release"
    exit 1
fi

echo "Latest version: $LATEST"

# Download URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST/$BINARY_NAME-$PLATFORM"

if [ "$PLATFORM" = "windows-amd64" ]; then
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST/$BINARY_NAME-windows-amd64.exe"
fi

# Download binary
echo "Downloading $BINARY_NAME for $PLATFORM..."
TMP_FILE=$(mktemp)
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"

# Make executable
chmod +x "$TMP_FILE"

# Install
echo "Installing to $INSTALL_DIR/$BINARY_NAME..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
else
    sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
fi

echo ""
echo "Installed $BINARY_NAME $LATEST to $INSTALL_DIR/$BINARY_NAME"
echo ""
echo "Quick setup for Claude Code:"
echo "  claude mcp add mediawiki $BINARY_NAME -e MEDIAWIKI_URL=\"https://your-wiki.com/api.php\""
echo ""
echo "See https://github.com/$REPO for full documentation."
