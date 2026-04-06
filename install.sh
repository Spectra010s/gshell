#!/bin/bash

set -e

INSTALL_DIR="${PREFIX:-/usr/local}/bin"
COMPLETION_DIR="${PREFIX:-/usr/local}/etc/bash_completion.d"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo ""
echo "  Installing gshell..."
echo ""

# Helper to open a URL in the user's browser if possible
open_url() {
    url="$1"
    if command -v xdg-open >/dev/null 2>&1; then
        xdg-open "$url" >/dev/null 2>&1 || true
    elif command -v open >/dev/null 2>&1; then
        open "$url" >/dev/null 2>&1 || true
    elif command -v start >/dev/null 2>&1; then
        start "$url" >/dev/null 2>&1 || true
    else
        echo "  Visit: $url"
    fi
}

# Check gcloud is installed (interactive)
if ! command -v gcloud &>/dev/null; then
    echo "  ⚠️  gcloud is not installed."
    echo "  Install it from: https://cloud.google.com/sdk/docs/install"
    read -p "Open the installation page in your browser now? [y/N] " resp
    if [ "$resp" = "y" ] || [ "$resp" = "Y" ]; then
        open_url "https://cloud.google.com/sdk/docs/install"
        echo "  Opened browser. Follow the instructions and re-run this installer."
    else
        echo "  Please install the Cloud SDK and re-run this script."
    fi
    exit 1
fi

echo "  ✓ gcloud found: $(gcloud --version | head -n1)"

# Check for cloud-shell alpha subcommand and offer to install components
if ! gcloud alpha cloud-shell --help >/dev/null 2>&1; then
    echo "  ⚠️  Cloud Shell alpha subcommand not available. This script recommends installing the 'alpha' components."
    read -p "Run 'gcloud components install alpha' now? [y/N] " resp
    if [ "$resp" = "y" ] || [ "$resp" = "Y" ]; then
        echo "  Installing alpha components in background..."
        gcloud components install alpha >/dev/null 2>&1 &
        echo "  Installation started in background; re-run installer after it completes."
        exit 0
    else
        echo "  You can enable it later with: gcloud components install alpha"
    fi
fi

# Create install dir if needed
mkdir -p "$INSTALL_DIR"
mkdir -p "$COMPLETION_DIR"

# Copy gshell
cp "$SCRIPT_DIR/gshell" "$INSTALL_DIR/gshell"
chmod +x "$INSTALL_DIR/gshell"
echo "  ✓ gshell installed to $INSTALL_DIR/gshell"

# Copy completion script
cp "$SCRIPT_DIR/gshell-completion.bash" "$COMPLETION_DIR/gshell-completion.bash"
echo "  ✓ completion script installed to $COMPLETION_DIR/gshell-completion.bash"

# Source completion in .bashrc if not already there
if ! grep -q "gshell-completion" "$HOME/.bashrc"; then
    echo "" >> "$HOME/.bashrc"
    echo "# gshell completion" >> "$HOME/.bashrc"
    echo "source \"$COMPLETION_DIR/gshell-completion.bash\"" >> "$HOME/.bashrc"
    echo "  ✓ Completion sourced in .bashrc"
fi

echo ""
echo "  gshell installed successfully!"
echo "  Run: source ~/.bashrc or open a new terminal to get started."
echo "  Then type: gshell --help"
echo ""
