#!/usr/bin/env bash
#
# Test harness for pgbox releases using GitHub CLI
# Usage: ./scripts/test-release.sh [--tag TAG] [--dry-run]
#
set -euo pipefail

REPO="ahacop/pgbox"
TAG="${TAG:-}"
DRY_RUN="${DRY_RUN:-false}"
TEMP_DIR=""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { printf "${BLUE}[INFO]${NC} %s\n" "$1"; }
print_success() { printf "${GREEN}[PASS]${NC} %s\n" "$1"; }
print_error() { printf "${RED}[FAIL]${NC} %s\n" "$1" >&2; }
print_warning() { printf "${YELLOW}[WARN]${NC} %s\n" "$1"; }

cleanup() {
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
}
trap cleanup EXIT

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Test harness for pgbox releases using GitHub CLI"
    echo ""
    echo "Options:"
    echo "  --tag TAG    Release tag to test (default: latest)"
    echo "  --dry-run    Skip actual installation tests"
    echo "  -h, --help   Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                        # Test latest release"
    echo "  $0 --tag v0.2.2           # Test specific release"
    echo "  $0 --tag v0.2.2 --dry-run # Check assets without installing"
}

# Parse arguments
while [ $# -gt 0 ]; do
    case "$1" in
        --tag)
            TAG="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN="true"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown argument: $1"
            usage
            exit 1
            ;;
    esac
done

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."

    if ! command -v gh &> /dev/null; then
        print_error "GitHub CLI (gh) is required but not installed"
        exit 1
    fi

    if ! gh auth status &> /dev/null; then
        print_error "GitHub CLI is not authenticated. Run 'gh auth login'"
        exit 1
    fi

    print_success "Prerequisites check passed"
}

# Get release info
get_release_info() {
    print_info "Fetching release information..."

    if [ -z "$TAG" ]; then
        TAG=$(gh release view --repo "$REPO" --json tagName -q '.tagName' 2>/dev/null || echo "")
        if [ -z "$TAG" ]; then
            print_error "No releases found and no tag specified"
            exit 1
        fi
        print_info "Using latest release: $TAG"
    fi

    # Verify release exists
    if ! gh release view "$TAG" --repo "$REPO" &> /dev/null; then
        print_error "Release $TAG not found"
        exit 1
    fi

    print_success "Release $TAG found"
}

# Test 1: Verify release assets exist
test_release_assets() {
    print_info "Testing release assets..."

    local version="${TAG#v}"
    local expected_assets=(
        "pgbox_${version}_Linux_x86_64.tar.gz"
        "pgbox_${version}_Linux_arm64.tar.gz"
        "pgbox_${version}_Darwin_x86_64.tar.gz"
        "pgbox_${version}_Darwin_arm64.tar.gz"
        "checksums.txt"
    )

    local assets
    assets=$(gh release view "$TAG" --repo "$REPO" --json assets -q '.assets[].name')

    local missing=0
    for expected in "${expected_assets[@]}"; do
        if echo "$assets" | grep -q "^${expected}$"; then
            print_success "Asset found: $expected"
        else
            print_error "Asset missing: $expected"
            missing=$((missing + 1))
        fi
    done

    if [ "$missing" -gt 0 ]; then
        print_error "$missing assets missing"
        return 1
    fi

    print_success "All release assets present"
}

# Test 2: Download and verify binary
test_binary_download() {
    print_info "Testing binary download and execution..."

    TEMP_DIR=$(mktemp -d)

    # Detect current platform
    local os arch archive_os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    case "$os" in
        linux) archive_os="Linux" ;;
        darwin) archive_os="Darwin" ;;
        *) print_warning "Unsupported OS: $os"; return 0 ;;
    esac

    case "$arch" in
        x86_64|amd64) arch="x86_64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) print_warning "Unsupported arch: $arch"; return 0 ;;
    esac

    local version="${TAG#v}"
    local archive_name="pgbox_${version}_${archive_os}_${arch}.tar.gz"

    print_info "Downloading $archive_name..."
    gh release download "$TAG" --repo "$REPO" --pattern "$archive_name" --dir "$TEMP_DIR"

    print_info "Extracting..."
    tar -xzf "$TEMP_DIR/$archive_name" -C "$TEMP_DIR"

    if [ ! -f "$TEMP_DIR/pgbox" ]; then
        print_error "Binary not found in archive"
        return 1
    fi

    chmod +x "$TEMP_DIR/pgbox"

    print_info "Testing binary execution..."
    local version_output
    version_output=$("$TEMP_DIR/pgbox" --version 2>&1)

    if echo "$version_output" | grep -q "${version}"; then
        print_success "Binary version matches tag: $version_output"
    else
        print_warning "Version mismatch. Expected ${version}, got: $version_output"
    fi

    print_success "Binary download and execution test passed"
}

# Test 3: Verify checksums
test_checksums() {
    print_info "Testing checksums..."

    if [ -z "$TEMP_DIR" ] || [ ! -d "$TEMP_DIR" ]; then
        TEMP_DIR=$(mktemp -d)
    fi

    gh release download "$TAG" --repo "$REPO" --pattern "checksums.txt" --dir "$TEMP_DIR" --clobber

    if [ ! -f "$TEMP_DIR/checksums.txt" ]; then
        print_error "checksums.txt not found"
        return 1
    fi

    # Verify at least one downloaded archive
    local archives
    archives=$(find "$TEMP_DIR" -name "*.tar.gz" 2>/dev/null || true)

    if [ -n "$archives" ]; then
        cd "$TEMP_DIR"
        if sha256sum --check checksums.txt --ignore-missing 2>/dev/null; then
            print_success "Checksum verification passed"
        elif shasum -a 256 --check checksums.txt --ignore-missing 2>/dev/null; then
            print_success "Checksum verification passed"
        else
            print_error "Checksum verification failed"
            cd - > /dev/null
            return 1
        fi
        cd - > /dev/null
    else
        print_warning "No archives to verify checksums against"
    fi
}

# Test 4: Test install script
test_install_script() {
    print_info "Testing install script availability..."

    local install_url="https://raw.githubusercontent.com/$REPO/$TAG/install.sh"

    if curl -sSL --fail "$install_url" -o /dev/null; then
        print_success "Install script accessible at $install_url"
    else
        print_error "Install script not accessible"
        return 1
    fi

    if [ "$DRY_RUN" = "false" ]; then
        print_info "Testing install script execution..."

        if [ -z "$TEMP_DIR" ] || [ ! -d "$TEMP_DIR" ]; then
            TEMP_DIR=$(mktemp -d)
        fi

        # Download install script
        curl -sSL "$install_url" -o "$TEMP_DIR/install.sh"
        chmod +x "$TEMP_DIR/install.sh"

        # Run with force flag and custom install dir
        INSTALL_DIR="$TEMP_DIR/bin" PGBOX_VERSION="$TAG" "$TEMP_DIR/install.sh" --force

        if [ -f "$TEMP_DIR/bin/pgbox" ]; then
            print_success "Install script execution passed"
        else
            print_error "Install script did not create binary"
            return 1
        fi
    else
        print_info "Skipping install script execution (dry-run mode)"
    fi
}

# Test 5: Test go install
test_go_install() {
    print_info "Testing go install availability..."

    if ! command -v go &> /dev/null; then
        print_warning "Go not available, skipping go install test"
        return 0
    fi

    if [ "$DRY_RUN" = "false" ]; then
        print_info "Testing go install command..."

        if [ -z "$TEMP_DIR" ] || [ ! -d "$TEMP_DIR" ]; then
            TEMP_DIR=$(mktemp -d)
        fi

        if GOBIN="$TEMP_DIR/gobin" go install "github.com/$REPO@$TAG" 2>&1; then
            if [ -f "$TEMP_DIR/gobin/pgbox" ]; then
                print_success "go install succeeded"
            else
                print_warning "go install completed but binary not found"
            fi
        else
            print_warning "go install failed (may take time for new releases to propagate)"
        fi
    else
        print_info "Skipping go install execution (dry-run mode)"
    fi
}

# Test 6: Test nix flake
test_nix_flake() {
    print_info "Testing nix flake..."

    if ! command -v nix &> /dev/null; then
        print_warning "Nix not available, skipping flake test"
        return 0
    fi

    if [ "$DRY_RUN" = "false" ]; then
        print_info "Testing nix run command..."

        if timeout 120 nix run "github:$REPO/$TAG" -- --version 2>&1; then
            print_success "nix flake test passed"
        else
            print_warning "nix flake test failed (may be expected for some releases)"
        fi
    else
        print_info "Skipping nix flake execution (dry-run mode)"
    fi
}

# Main test runner
main() {
    echo "=========================================="
    echo "pgbox Release Test Harness"
    echo "=========================================="
    echo ""

    check_prerequisites
    get_release_info

    echo ""
    echo "Running tests for release: $TAG"
    echo "------------------------------------------"

    local failures=0

    test_release_assets || failures=$((failures + 1))
    test_binary_download || failures=$((failures + 1))
    test_checksums || failures=$((failures + 1))
    test_install_script || failures=$((failures + 1))
    test_go_install || true  # Don't fail on go install issues
    test_nix_flake || true   # Don't fail on nix issues

    echo ""
    echo "=========================================="
    if [ "$failures" -eq 0 ]; then
        print_success "All critical tests passed!"
        exit 0
    else
        print_error "$failures critical test(s) failed"
        exit 1
    fi
}

main "$@"
