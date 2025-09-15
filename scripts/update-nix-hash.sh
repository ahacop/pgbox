#!/usr/bin/env bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Updating Nix vendorHash...${NC}"

# Check if nix is available
if ! command -v nix &> /dev/null; then
    echo -e "${RED}Error: nix command not found${NC}"
    exit 1
fi

# Backup the original flake.nix
cp flake.nix flake.nix.bak

# Set vendorHash to empty string to force error with correct hash
sed -i 's/vendorHash = ".*";/vendorHash = "";/' flake.nix

# Try to build and capture the output
echo "Running nix build to get correct hash..."
BUILD_OUTPUT=$(nix build 2>&1 || true)

# Extract the correct hash from the error message
CORRECT_HASH=$(echo "$BUILD_OUTPUT" | grep "got:" | sed 's/.*got:[[:space:]]*//')

if [ -z "$CORRECT_HASH" ]; then
    echo -e "${RED}Error: Could not extract hash from nix build output${NC}"
    echo "Build output:"
    echo "$BUILD_OUTPUT"
    # Restore backup
    mv flake.nix.bak flake.nix
    exit 1
fi

# Update flake.nix with the correct hash
sed -i "s/vendorHash = \"\";/vendorHash = \"$CORRECT_HASH\";/" flake.nix

# Remove backup
rm -f flake.nix.bak

echo -e "${GREEN}✓ vendorHash updated to: $CORRECT_HASH${NC}"

# Verify the build works now
echo "Verifying build..."
if nix build --no-link 2>/dev/null; then
    echo -e "${GREEN}✓ Nix build successful${NC}"
else
    echo -e "${YELLOW}Warning: Nix build still failing. You may need to check other issues.${NC}"
fi