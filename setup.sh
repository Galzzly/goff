#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
cd "$SCRIPT_DIR"

# Run the setup command
go run ./template/cmd/setup

# Clean up build artifacts
rm -rf template/cmd template/internal template/readme

# Remove this setup script
rm "$0"

echo "Setup complete. You can now use the 'goff' binary."
