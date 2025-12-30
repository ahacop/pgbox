#!/usr/bin/env bash
#
# Prepare a new pgbox release
# Usage: ./scripts/prepare-release.sh v0.3.0
#
set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { printf "${BLUE}[INFO]${NC} %s\n" "$1"; }
print_success() { printf "${GREEN}[OK]${NC} %s\n" "$1"; }
print_error() { printf "${RED}[ERROR]${NC} %s\n" "$1" >&2; }
print_warning() { printf "${YELLOW}[WARN]${NC} %s\n" "$1"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

usage() {
    echo "Usage: $0 <version>"
    echo ""
    echo "Prepare a new pgbox release by updating VERSION, vendorHash, and creating a git tag."
    echo ""
    echo "Arguments:"
    echo "  version    Release version (e.g., v0.3.0 or 0.3.0)"
    echo ""
    echo "Examples:"
    echo "  $0 v0.3.0"
    echo "  $0 0.3.0"
    echo ""
    echo "After running this script, push the tag with:"
    echo "  git push origin <tag>"
}

# Validate version format
validate_version() {
    local version="$1"
    # Strip leading 'v' if present for validation
    local ver="${version#v}"

    if ! echo "$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$'; then
        print_error "Invalid version format: $version"
        echo "Expected format: X.Y.Z or vX.Y.Z (e.g., 0.3.0 or v0.3.0)"
        exit 1
    fi
}

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."

    # Check we're in the project root
    if [ ! -f "$PROJECT_ROOT/go.mod" ]; then
        print_error "Not in pgbox project root"
        exit 1
    fi

    # Check git status
    if [ -n "$(git status --porcelain)" ]; then
        print_warning "Working directory has uncommitted changes"
        echo "Consider committing or stashing changes first."
        read -rp "Continue anyway? [y/N] " response
        case "$response" in
            [yY][eE][sS]|[yY]) ;;
            *) echo "Aborted."; exit 0 ;;
        esac
    fi

    # Check nix is available
    if ! command -v nix &> /dev/null; then
        print_error "nix command not found (required for vendorHash update)"
        exit 1
    fi

    print_success "Prerequisites check passed"
}

# Update VERSION file
update_version_file() {
    local version="$1"
    # Strip leading 'v' for VERSION file
    local ver="${version#v}"

    print_info "Updating VERSION file to $ver..."
    echo "$ver" > "$PROJECT_ROOT/VERSION"
    print_success "VERSION file updated"
}

# Update vendorHash in flake.nix
update_vendor_hash() {
    print_info "Updating vendorHash in flake.nix..."

    cd "$PROJECT_ROOT"

    # Backup the original flake.nix
    cp flake.nix flake.nix.bak

    # Set vendorHash to empty string to force error with correct hash
    sed -i 's|vendorHash = ".*";|vendorHash = "";|' flake.nix

    # Try to build and capture the output
    print_info "Running nix build to get correct hash (this may take a moment)..."
    BUILD_OUTPUT=$(nix build 2>&1 || true)

    # Extract the correct hash from the error message
    CORRECT_HASH=$(echo "$BUILD_OUTPUT" | grep "got:" | sed 's/.*got:[[:space:]]*//' | tr -d ' ')

    if [ -z "$CORRECT_HASH" ]; then
        print_error "Could not extract hash from nix build output"
        echo "Build output:"
        echo "$BUILD_OUTPUT"
        # Restore backup
        mv flake.nix.bak flake.nix
        exit 1
    fi

    # Update flake.nix with the correct hash
    sed -i "s|vendorHash = \"\";|vendorHash = \"$CORRECT_HASH\";|" flake.nix

    # Remove backup
    rm -f flake.nix.bak

    print_success "vendorHash updated to: $CORRECT_HASH"

    # Verify the build works
    print_info "Verifying nix build..."
    if nix build --no-link 2>/dev/null; then
        print_success "Nix build successful"
    else
        print_warning "Nix build still failing - check for other issues"
    fi
}

# Run tests
run_tests() {
    print_info "Running tests..."
    cd "$PROJECT_ROOT"

    if make test; then
        print_success "Tests passed"
    else
        print_error "Tests failed"
        exit 1
    fi
}

# Create git tag
create_git_tag() {
    local version="$1"
    # Ensure version starts with 'v' for git tag
    local tag="${version}"
    if [[ ! "$tag" =~ ^v ]]; then
        tag="v$tag"
    fi

    print_info "Creating git tag: $tag"

    # Check if tag already exists
    if git rev-parse "$tag" &>/dev/null; then
        print_error "Tag $tag already exists"
        exit 1
    fi

    # Stage changes
    git add VERSION flake.nix

    # Commit if there are changes
    if ! git diff --cached --quiet; then
        git commit -m "chore: Prepare release $tag"
        print_success "Changes committed"
    else
        print_info "No changes to commit"
    fi

    # Create tag
    git tag "$tag"
    print_success "Tag $tag created"

    echo ""
    echo "=========================================="
    print_success "Release $tag prepared!"
    echo "=========================================="
    echo ""
    echo "Next steps:"
    echo "  1. Review the changes: git show HEAD"
    echo "  2. Push the commit:    git push origin main"
    echo "  3. Push the tag:       git push origin $tag"
    echo ""
    echo "The GitHub Actions workflow will automatically create the release."
}

# Main
main() {
    if [ $# -lt 1 ]; then
        usage
        exit 1
    fi

    local version="$1"

    if [ "$version" = "-h" ] || [ "$version" = "--help" ]; then
        usage
        exit 0
    fi

    echo "=========================================="
    echo "pgbox Release Preparation"
    echo "=========================================="
    echo ""

    validate_version "$version"
    check_prerequisites

    echo ""
    print_info "Preparing release: $version"
    echo ""

    update_version_file "$version"
    update_vendor_hash
    run_tests
    create_git_tag "$version"
}

main "$@"
