#!/bin/bash

# Fly.io CLI Installation Script
# Automatically detects the best installation method for your system

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}\n"
}

# Detect operating system
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macOS"
        # Detect architecture
        if [[ $(uname -m) == "arm64" ]]; then
            ARCH="Apple Silicon"
        else
            ARCH="Intel"
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        OS="Linux"
        ARCH=$(uname -m)
    elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
        OS="Windows"
        ARCH=$(uname -m)
    else
        OS="Unknown"
        ARCH="Unknown"
    fi
}

# Check if flyctl is already installed
check_existing_installation() {
    print_header "Checking Existing Installation"
    
    if command -v flyctl &> /dev/null; then
        local version=$(flyctl version 2>/dev/null | head -1)
        print_success "flyctl is already installed: $version"
        
        # Ask if user wants to continue anyway
        echo ""
        read -p "Do you want to reinstall or update flyctl? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_status "Installation cancelled by user"
            exit 0
        fi
    else
        print_status "flyctl not found - proceeding with installation"
    fi
}

# Check if Homebrew is available
check_homebrew() {
    if command -v brew &> /dev/null; then
        HOMEBREW_AVAILABLE=true
        print_status "Homebrew detected"
    else
        HOMEBREW_AVAILABLE=false
        print_status "Homebrew not available"
    fi
}

# Install using direct download (recommended method)
install_direct() {
    print_header "Installing flyctl via Direct Download"
    
    print_status "Downloading and installing flyctl..."
    
    if [[ "$OS" == "Windows" ]]; then
        print_error "Direct download method not supported on Windows"
        print_status "Please use PowerShell method: iwr https://fly.io/install.ps1 -useb | iex"
        exit 1
    fi
    
    # Download and install
    if curl -L https://fly.io/install.sh | sh; then
        print_success "flyctl installed successfully via direct download"
        
        # Check if flyctl is in PATH
        if ! command -v flyctl &> /dev/null; then
            print_warning "flyctl may not be in your PATH"
            print_status "Adding flyctl to PATH..."
            
            # Add to PATH in shell profile
            local shell_profile=""
            if [[ -f "$HOME/.zshrc" ]]; then
                shell_profile="$HOME/.zshrc"
            elif [[ -f "$HOME/.bashrc" ]]; then
                shell_profile="$HOME/.bashrc"
            elif [[ -f "$HOME/.bash_profile" ]]; then
                shell_profile="$HOME/.bash_profile"
            fi
            
            if [[ -n "$shell_profile" ]]; then
                echo 'export PATH="$HOME/.fly/bin:$PATH"' >> "$shell_profile"
                print_success "Added flyctl to PATH in $shell_profile"
                print_warning "Please run 'source $shell_profile' or restart your terminal"
                
                # Temporarily add to current session
                export PATH="$HOME/.fly/bin:$PATH"
            else
                print_warning "Could not detect shell profile. Please manually add $HOME/.fly/bin to your PATH"
            fi
        fi
        
        return 0
    else
        print_error "Direct download installation failed"
        return 1
    fi
}

# Install using Homebrew
install_homebrew() {
    print_header "Installing flyctl via Homebrew"
    
    if ! $HOMEBREW_AVAILABLE; then
        print_error "Homebrew is not available"
        return 1
    fi
    
    print_status "Installing flyctl using Homebrew..."
    
    if brew install flyctl; then
        print_success "flyctl installed successfully via Homebrew"
        return 0
    else
        print_error "Homebrew installation failed"
        return 1
    fi
}

# Install using manual download from GitHub
install_manual() {
    print_header "Manual Installation from GitHub"
    
    print_status "This method requires manual download from GitHub releases"
    print_status "Visit: https://github.com/superfly/flyctl/releases"
    
    if [[ "$OS" == "macOS" ]]; then
        if [[ "$ARCH" == "Apple Silicon" ]]; then
            print_status "Download: flyctl_*_macOS_arm64.tar.gz"
        else
            print_status "Download: flyctl_*_macOS_x86_64.tar.gz"
        fi
    elif [[ "$OS" == "Linux" ]]; then
        print_status "Download: flyctl_*_Linux_x86_64.tar.gz"
    fi
    
    print_status "After download, extract and move to /usr/local/bin/"
    print_status "Example commands:"
    echo "  tar -xzf flyctl_*.tar.gz"
    echo "  sudo mv flyctl /usr/local/bin/"
    echo "  chmod +x /usr/local/bin/flyctl"
    
    return 1  # This method requires manual intervention
}

# Verify installation
verify_installation() {
    print_header "Verifying Installation"
    
    if command -v flyctl &> /dev/null; then
        local version=$(flyctl version 2>/dev/null | head -1)
        print_success "flyctl is installed and working: $version"
        
        # Test basic functionality
        print_status "Testing flyctl functionality..."
        if flyctl help &> /dev/null; then
            print_success "flyctl is functioning correctly"
        else
            print_warning "flyctl installed but may have issues"
        fi
        
        return 0
    else
        print_error "flyctl installation verification failed"
        print_error "flyctl command not found in PATH"
        return 1
    fi
}

# Show next steps
show_next_steps() {
    print_header "Next Steps"
    
    echo "Now that flyctl is installed, you can:"
    echo ""
    echo "1. Login to Fly.io:"
    echo "   flyctl auth login"
    echo ""
    echo "2. Deploy your application:"
    echo "   ./deploy.sh"
    echo ""
    echo "3. Check flyctl version:"
    echo "   flyctl version"
    echo ""
    echo "4. Get help:"
    echo "   flyctl help"
    echo ""
    print_success "Installation complete! 🚀"
}

# Main installation logic
main() {
    echo -e "${GREEN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                    Fly.io CLI Installer                      ║"
    echo "║              Automatic flyctl Installation                   ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    # Detect system
    detect_os
    print_status "Detected OS: $OS ($ARCH)"
    
    # Check existing installation
    check_existing_installation
    
    # Check available installation methods
    check_homebrew
    
    # Choose installation method
    print_header "Choosing Installation Method"
    
    local installation_success=false
    
    # Try direct download first (recommended)
    if [[ "$OS" != "Windows" ]]; then
        print_status "Attempting direct download installation (recommended)..."
        if install_direct; then
            installation_success=true
        fi
    fi
    
    # Fall back to Homebrew if available and direct download failed
    if ! $installation_success && $HOMEBREW_AVAILABLE; then
        print_status "Falling back to Homebrew installation..."
        if install_homebrew; then
            installation_success=true
        fi
    fi
    
    # If all automated methods failed, show manual instructions
    if ! $installation_success; then
        print_warning "Automated installation methods failed"
        install_manual
        
        echo ""
        print_status "Alternative installation commands:"
        
        if [[ "$OS" == "macOS" ]]; then
            echo ""
            echo "Install Homebrew first, then flyctl:"
            echo '  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"'
            echo "  brew install flyctl"
            echo ""
            echo "Or use MacPorts:"
            echo "  sudo port install flyctl"
        elif [[ "$OS" == "Linux" ]]; then
            echo ""
            echo "Try the direct download again:"
            echo "  curl -L https://fly.io/install.sh | sh"
            echo ""
            echo "Or use Nix:"
            echo "  nix-env -iA nixpkgs.flyctl"
        elif [[ "$OS" == "Windows" ]]; then
            echo ""
            echo "PowerShell (recommended):"
            echo "  iwr https://fly.io/install.ps1 -useb | iex"
            echo ""
            echo "Chocolatey:"
            echo "  choco install flyctl"
            echo ""
            echo "Scoop:"
            echo "  scoop install flyctl"
        fi
        
        exit 1
    fi
    
    # Verify installation
    if verify_installation; then
        show_next_steps
    else
        print_error "Installation verification failed"
        exit 1
    fi
}

# Handle script interruption
trap 'echo -e "\n${YELLOW}Installation interrupted by user${NC}"; exit 1' INT

# Run main function
main "$@"