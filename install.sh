#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}Installing mump2p CLI...${NC}"

print_error() {
    echo -e "${RED}Error: $1${NC}"
}

print_success() {
    echo -e "${GREEN}Success: $1${NC}"
}

print_info() {
    echo -e "${BLUE}Info: $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}Warning: $1${NC}"
}

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

print_info "Detected OS: $OS"
print_info "Detected Architecture: $ARCH"

# Map OS names to binary names
case $OS in
    linux*)     BINARY_NAME="mump2p-linux" ;;
    darwin*)    BINARY_NAME="mump2p-mac" ;;
    *)          print_error "Unsupported OS: $OS. Supported: Linux, macOS"
                exit 1 ;;
esac

# Check if curl or wget is available
if command -v curl >/dev/null 2>&1; then
    DOWNLOADER="curl"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOADER="wget"
else
    print_error "Neither curl nor wget found. Please install one of them."
    exit 1
fi

# Get latest release info
print_info "Fetching latest release information..."
if [ "$DOWNLOADER" = "curl" ]; then
    RELEASE_INFO=$(curl -s https://api.github.com/repos/getoptimum/mump2p-cli/releases/latest)
else
    RELEASE_INFO=$(wget -qO- https://api.github.com/repos/getoptimum/mump2p-cli/releases/latest)
fi

LATEST=$(echo "$RELEASE_INFO" | grep '"tag_name":' | cut -d '"' -f 4)

if [ -z "$LATEST" ]; then
    print_error "Failed to fetch latest release information"
    echo "Please check your internet connection or try manual installation:"
    echo "https://github.com/getoptimum/mump2p-cli/releases"
    exit 1
fi

print_success "Latest version: $LATEST"

# Construct download URL
URL="https://github.com/getoptimum/mump2p-cli/releases/download/$LATEST/$BINARY_NAME"

print_info "Downloading: $BINARY_NAME"
print_info "From: $URL"

# Download binary
if [ "$DOWNLOADER" = "curl" ]; then
    if ! curl -L -f -o mump2p "$URL"; then
        print_error "Download failed. Binary might not exist for your platform."
        echo "Available releases: https://github.com/getoptimum/mump2p-cli/releases/latest"
        exit 1
    fi
else
    if ! wget -O mump2p "$URL"; then
        print_error "Download failed. Binary might not exist for your platform."
        echo "Available releases: https://github.com/getoptimum/mump2p-cli/releases/latest"
        exit 1
    fi
fi

# Make executable
chmod +x mump2p

print_success "Binary downloaded and made executable"

# Verify installation
print_info "Verifying installation..."
if ./mump2p version >/dev/null 2>&1; then
    VERSION=$(./mump2p version 2>/dev/null || echo "unknown")
    print_success "mump2p installed successfully!"
    echo ""
    echo -e "${GREEN}Installation Details:${NC}"
    echo "   Location: $(pwd)/mump2p"
    echo "   Version: $VERSION"
    echo ""
    echo -e "${BLUE}Next Steps:${NC}"
    echo "   ./mump2p help       # View all commands"
    echo "   ./mump2p login      # Authenticate and start using"
    echo ""
    echo -e "${YELLOW}Tip: Add to PATH for global access${NC}"
    echo "   sudo mv mump2p /usr/local/bin/"
else
    print_error "Installation verification failed"
    echo "The binary was downloaded but doesn't seem to work on your system."
    echo "Please try manual installation or report this issue."
    exit 1
fi
