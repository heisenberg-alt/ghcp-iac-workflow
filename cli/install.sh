#!/bin/bash

# =============================================================================
# IaC Lab Installer
# =============================================================================
# Cross-platform installer for the IaC Lab CLI
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'

echo -e "${CYAN}${BOLD}"
cat << 'EOF'

    ██╗ █████╗  ██████╗    ██╗      █████╗ ██████╗ 
    ██║██╔══██╗██╔════╝    ██║     ██╔══██╗██╔══██╗
    ██║███████║██║         ██║     ███████║██████╔╝
    ██║██╔══██║██║         ██║     ██╔══██║██╔══██╗
    ██║██║  ██║╚██████╗    ███████╗██║  ██║██████╔╝
    ╚═╝╚═╝  ╚═╝ ╚═════╝    ╚══════╝╚═╝  ╚═╝╚═════╝ 
                                                    
              🔧 Installer 🔧

EOF
echo -e "${NC}"

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Darwin*)    echo "macos" ;;
        Linux*)     echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)          echo "unknown" ;;
    esac
}

OS=$(detect_os)
echo -e "${CYAN}Detected OS: ${WHITE}$OS${NC}"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LAB_ROOT="$(dirname "$SCRIPT_DIR")"

echo -e "${CYAN}Lab root: ${WHITE}$LAB_ROOT${NC}"
echo ""

# Make the main script executable
chmod +x "$SCRIPT_DIR/iac-lab.sh"

# Installation options
install_local() {
    echo -e "${YELLOW}Installing locally...${NC}"
    
    # Create a wrapper script in a common location
    local install_dir=""
    local shell_rc=""
    
    case $OS in
        macos|linux)
            install_dir="$HOME/.local/bin"
            if [[ -f "$HOME/.zshrc" ]]; then
                shell_rc="$HOME/.zshrc"
            elif [[ -f "$HOME/.bashrc" ]]; then
                shell_rc="$HOME/.bashrc"
            fi
            ;;
        windows)
            install_dir="$HOME/bin"
            shell_rc="$HOME/.bashrc"
            ;;
    esac
    
    mkdir -p "$install_dir"
    
    # Create wrapper script
    cat > "$install_dir/iac-lab" << EOF
#!/bin/bash
exec "$SCRIPT_DIR/iac-lab.sh" "\$@"
EOF
    chmod +x "$install_dir/iac-lab"
    
    # Add to PATH if needed
    if [[ -n "$shell_rc" ]] && ! grep -q "$install_dir" "$shell_rc" 2>/dev/null; then
        echo "" >> "$shell_rc"
        echo "# IaC Lab CLI" >> "$shell_rc"
        echo "export PATH=\"\$PATH:$install_dir\"" >> "$shell_rc"
        echo -e "${YELLOW}Added $install_dir to PATH in $shell_rc${NC}"
    fi
    
    echo -e "${GREEN}✅ Installation complete!${NC}"
    echo ""
    echo -e "${WHITE}To start using the IaC Lab:${NC}"
    echo -e "${CYAN}  1. Restart your terminal or run: source $shell_rc${NC}"
    echo -e "${CYAN}  2. Run: iac-lab${NC}"
}

install_symlink() {
    echo -e "${YELLOW}Creating symlink...${NC}"
    
    local link_dir="/usr/local/bin"
    
    if [[ ! -w "$link_dir" ]]; then
        echo -e "${YELLOW}Requires sudo to install to $link_dir${NC}"
        sudo ln -sf "$SCRIPT_DIR/iac-lab.sh" "$link_dir/iac-lab"
    else
        ln -sf "$SCRIPT_DIR/iac-lab.sh" "$link_dir/iac-lab"
    fi
    
    echo -e "${GREEN}✅ Symlink created!${NC}"
    echo -e "${WHITE}Run 'iac-lab' to start.${NC}"
}

# Show menu
echo -e "${WHITE}${BOLD}Select installation method:${NC}"
echo ""
echo -e "  ${CYAN}[1]${NC} Local install (recommended) - Adds to ~/.local/bin"
echo -e "  ${CYAN}[2]${NC} System symlink - Creates link in /usr/local/bin"
echo -e "  ${CYAN}[3]${NC} Just run now - No installation"
echo -e "  ${CYAN}[q]${NC} Cancel"
echo ""
echo -ne "${YELLOW}Select option: ${NC}"

read -r choice

case $choice in
    1)
        install_local
        ;;
    2)
        install_symlink
        ;;
    3)
        echo -e "${CYAN}Starting IaC Lab...${NC}"
        exec "$SCRIPT_DIR/iac-lab.sh"
        ;;
    q|Q)
        echo -e "${YELLOW}Installation cancelled.${NC}"
        exit 0
        ;;
    *)
        echo -e "${RED}Invalid option.${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}${BOLD}🎉 Setup complete! Happy learning!${NC}"
