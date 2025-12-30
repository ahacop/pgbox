#!/bin/sh
#
# pgbox installer script
# This script downloads and installs the pgbox binary for your platform
#

set -e

# Configuration
REPO_OWNER="ahacop"
REPO_NAME="pgbox"
BINARY_NAME="pgbox"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
FORCE="${PGBOX_FORCE:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_info() {
    printf "${BLUE}[INFO]${NC} %s\n" "$1"
}

print_success() {
    printf "${GREEN}[SUCCESS]${NC} %s\n" "$1"
}

print_error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1" >&2
}

print_warning() {
    printf "${YELLOW}[WARNING]${NC} %s\n" "$1"
}

# Detect OS
detect_os() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$OS" in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="darwin"
            ;;
        *)
            print_error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac
    echo "$OS"
}

# Detect architecture
detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64)
            ARCH="x86_64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    echo "$ARCH"
}

# Get latest release version
get_latest_version() {
    VERSION="${PGBOX_VERSION:-latest}"
    if [ "$VERSION" = "latest" ]; then
        VERSION=$(curl -sL "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest" | \
                  grep '"tag_name":' | \
                  sed -E 's/.*"([^"]+)".*/\1/')

        if [ -z "$VERSION" ]; then
            print_error "Failed to fetch latest version"
            exit 1
        fi
    fi
    echo "$VERSION"
}

# Download and install binary
install_binary() {
    local os=$1
    local arch=$2
    local version=$3

    # Construct download URL
    # Capitalize OS name for archive naming convention
    local os_cap
    case "$os" in
        linux)
            os_cap="Linux"
            ;;
        darwin)
            os_cap="Darwin"
            ;;
        *)
            os_cap="$os"
            ;;
    esac

    local archive_name="${BINARY_NAME}_${version#v}_${os_cap}_${arch}.tar.gz"
    local download_url="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$version/$archive_name"

    print_info "Downloading $BINARY_NAME $version for $os/$arch..."
    print_info "URL: $download_url"

    # Create temp directory
    TEMP_DIR=$(mktemp -d)
    trap "rm -rf $TEMP_DIR" EXIT

    # Download archive
    if ! curl -sL "$download_url" -o "$TEMP_DIR/$archive_name"; then
        print_error "Failed to download $BINARY_NAME"
        exit 1
    fi

    # Extract archive
    print_info "Extracting archive..."
    if ! tar -xzf "$TEMP_DIR/$archive_name" -C "$TEMP_DIR"; then
        print_error "Failed to extract archive"
        exit 1
    fi

    # Find the binary
    if [ ! -f "$TEMP_DIR/$BINARY_NAME" ]; then
        print_error "Binary not found in archive"
        exit 1
    fi

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        print_info "Creating install directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi

    # Install binary
    print_info "Installing $BINARY_NAME to $INSTALL_DIR..."
    if ! mv "$TEMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"; then
        print_error "Failed to install binary"
        exit 1
    fi

    # Make executable
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    print_success "$BINARY_NAME installed successfully!"
}

# Verify installation
verify_installation() {
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ] && [ -x "$INSTALL_DIR/$BINARY_NAME" ]; then
        VERSION_OUTPUT=$("$INSTALL_DIR/$BINARY_NAME" --version 2>&1 || true)
        print_success "Installation verified: $VERSION_OUTPUT"
        return 0
    else
        print_error "Installation verification failed"
        return 1
    fi
}

# Add to PATH instructions
print_path_instructions() {
    # Check if install dir is in PATH
    if echo "$PATH" | grep -q "$INSTALL_DIR"; then
        print_success "$INSTALL_DIR is already in your PATH"
    else
        print_warning "$INSTALL_DIR is not in your PATH"
        echo ""
        echo "Add it to your PATH by running one of these commands:"
        echo ""

        # Detect shell
        if [ -n "$BASH_VERSION" ]; then
            echo "  echo 'export PATH=\"\$PATH:$INSTALL_DIR\"' >> ~/.bashrc"
            echo "  source ~/.bashrc"
        elif [ -n "$ZSH_VERSION" ]; then
            echo "  echo 'export PATH=\"\$PATH:$INSTALL_DIR\"' >> ~/.zshrc"
            echo "  source ~/.zshrc"
        else
            echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
            echo ""
            echo "Add the above line to your shell's configuration file"
        fi
        echo ""
    fi
}

# Main installation process
main() {
    print_info "Starting $BINARY_NAME installation..."
    echo ""

    # Detect system
    OS=$(detect_os)
    ARCH=$(detect_arch)
    VERSION=$(get_latest_version)

    print_info "System detected: $OS/$ARCH"
    print_info "Version: $VERSION"
    print_info "Install directory: $INSTALL_DIR"
    echo ""

    # Check for existing installation
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        print_warning "$BINARY_NAME is already installed at $INSTALL_DIR/$BINARY_NAME"

        if [ "$FORCE" = "true" ]; then
            print_info "Force flag set, overwriting existing installation..."
        elif [ ! -t 0 ]; then
            # Non-interactive mode (piped input or CI environment)
            print_info "Non-interactive mode detected. Use --force to overwrite."
            exit 0
        else
            printf "Do you want to overwrite it? [y/N] "
            read -r response
            case "$response" in
                [yY][eE][sS]|[yY])
                    print_info "Overwriting existing installation..."
                    ;;
                *)
                    print_info "Installation cancelled"
                    exit 0
                    ;;
            esac
        fi
    fi

    # Install binary
    install_binary "$OS" "$ARCH" "$VERSION"

    # Verify installation
    if verify_installation; then
        echo ""
        print_success "Installation complete!"
        echo ""
        print_path_instructions
        echo ""
        echo "You can now use $BINARY_NAME by running:"
        echo "  $BINARY_NAME --help"
        echo ""
    else
        exit 1
    fi
}

# Parse arguments
while [ $# -gt 0 ]; do
    case "$1" in
        -f|--force)
            FORCE="true"
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -f, --force    Overwrite existing installation without prompting"
            echo "  -h, --help     Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  INSTALL_DIR    Installation directory (default: \$HOME/.local/bin)"
            echo "  PGBOX_VERSION  Version to install (default: latest)"
            echo "  PGBOX_FORCE    Set to 'true' to force overwrite (same as --force)"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Run main function
main