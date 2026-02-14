#!/usr/bin/env bash
#
# wt - Git Worktree Manager Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/bakerweb/wt/main/scripts/install.sh | bash
#
set -euo pipefail

REPO="bakerweb/wt"
INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="wt"
PATH_NOT_IN_PATH=false

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info()    { echo -e "${BLUE}[info]${NC} $*" >&2; }
success() { echo -e "${GREEN}[ok]${NC}   $*" >&2; }
warn()    { echo -e "${YELLOW}[warn]${NC} $*" >&2; }
error()   { echo -e "${RED}[err]${NC}  $*" >&2; }

# --- Dependency checks ---
check_deps() {
    local missing=()

    if ! command -v git &>/dev/null; then
        missing+=("git (>= 2.20)")
    else
        local git_version
        git_version=$(git --version | sed -E 's/^[^0-9]*([0-9]+\.[0-9]+).*/\1/')
        local git_major git_minor
        git_major=$(echo "$git_version" | cut -d. -f1)
        git_minor=$(echo "$git_version" | cut -d. -f2)
        if [ "$git_major" -lt 2 ] || ([ "$git_major" -eq 2 ] && [ "$git_minor" -lt 20 ]); then
            error "git >= 2.20 required (found $git_version)"
            missing+=("git (>= 2.20)")
        else
            success "git $git_version"
        fi
    fi

    if ! command -v curl &>/dev/null; then
        missing+=("curl")
    else
        success "curl"
    fi

    if ! command -v tar &>/dev/null && ! command -v unzip &>/dev/null; then
        missing+=("tar or unzip")
    else
        success "tar/unzip"
    fi

    if ! command -v bash &>/dev/null && ! command -v zsh &>/dev/null; then
        missing+=("bash or zsh")
    fi

    if (( ${#missing[@]} > 0 )); then
        error "Missing required dependencies:"
        for dep in "${missing[@]}"; do
            error "  - $dep"
        done
        exit 1
    fi

    success "All dependencies satisfied"
}

# --- Platform detection ---
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux*)
            os="linux"
            ;;
        Darwin*)
            os="darwin"
            ;;
        *)
            error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        arm64|aarch64)   arch="arm64" ;;
        *)
            error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

# --- Check WSL ---
check_wsl() {
    # Only check on Linux systems
    if [ "$(uname -s)" = "Linux" ] && [ -f /proc/version ]; then
        if grep -qi -E '(microsoft|wsl)' /proc/version 2>/dev/null; then
            info "Detected WSL environment"
        fi
    fi
}

# --- Get latest release ---
get_latest_version() {
    local url="https://api.github.com/repos/${REPO}/releases/latest"
    local version
    version=$(curl -fsSL "$url" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/')
    if [[ -z "$version" ]]; then
        error "Failed to fetch latest version from GitHub"
        exit 1
    fi
    echo "$version"
}

# --- Download and install ---
install() {
    local platform="$1"
    local version="$2"

    local download_url="https://github.com/${REPO}/releases/download/${version}/wt_${platform}.tar.gz"
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    info "Downloading wt ${version} for ${platform}..."
    if ! curl -fsSL "$download_url" -o "${tmp_dir}/wt.tar.gz"; then
        error "Download failed. Check that release ${version} exists for ${platform}"
        exit 1
    fi

    info "Extracting..."
    tar -xzf "${tmp_dir}/wt.tar.gz" -C "$tmp_dir"

    info "Installing to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR"
    mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

    success "Binary installed to ${INSTALL_DIR}/${BINARY_NAME}"
    echo "  Binary path: ${INSTALL_DIR}/${BINARY_NAME}"
}

# --- Check PATH ---
check_path() {
    if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
        PATH_NOT_IN_PATH=true
    fi
}

# --- Uninstall ---
uninstall() {
    info "Uninstalling wt..."

    if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
        rm -f "${INSTALL_DIR}/${BINARY_NAME}"
        success "Removed ${INSTALL_DIR}/${BINARY_NAME}"
    else
        warn "Binary not found at ${INSTALL_DIR}/${BINARY_NAME}"
    fi

    info "Note: Config directory ~/.wt/ was not removed. Delete it manually with: rm -rf ~/.wt"
    success "Uninstall complete"
}

# --- Main ---
main() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║   wt - Git Worktree Manager          ║${NC}"
    echo -e "${BLUE}║   Task-driven worktree management     ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════╝${NC}"
    echo ""

    # Handle --uninstall flag
    if [[ "${1:-}" == "--uninstall" ]]; then
        uninstall
        exit 0
    fi

    info "Checking dependencies..."
    check_deps
    echo ""

    info "Detecting platform..."
    local platform
    platform=$(detect_platform)
    if [[ "${platform:0:5}" == "linux" ]]; then
        check_wsl
    fi
    success "Platform: ${platform}"
    echo ""

    info "Fetching latest version..."
    local version
    version=$(get_latest_version)
    success "Version: ${version}"
    echo ""

    install "$platform" "$version"
    echo ""

    check_path
    
    success "Installation complete! 🎉"
    echo ""
    
    if [[ "$PATH_NOT_IN_PATH" == true ]]; then
        warn "\$HOME/.local/bin is not in your PATH"
        echo ""
        echo "  Add it with one of these commands:"
        echo ""
        echo "  For bash:"
        echo "    echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc && source ~/.bashrc"
        echo ""
        echo "  For zsh:"
        echo "    echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc && source ~/.zshrc"
        echo ""
        echo "  Or restart your terminal and try: wt --help"
    else
        echo "  wt is ready to use. Try:"
        echo "    wt --help"
    fi
    echo ""
}

main "$@"
