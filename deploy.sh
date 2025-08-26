#!/bin/bash

# Customer.io Pauser - Fly.io Deployment Script
# This script automates the deployment process including secrets management

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

# Check if flyctl is installed
check_flyctl() {
    print_header "Checking Prerequisites"
    
    if ! command -v flyctl &> /dev/null; then
        print_error "flyctl is not installed or not in PATH"
        echo ""
        echo "🚀 Quick Installation Options:"
        echo ""
        echo "1. Use our automated installer (recommended):"
        echo "   ./install-flyctl.sh"
        echo ""
        echo "2. Direct download (no Homebrew required):"
        echo "   curl -L https://fly.io/install.sh | sh"
        echo ""
        echo "3. Using Homebrew (if you have it installed):"
        echo "   brew install flyctl"
        echo ""
        echo "4. Install Homebrew first, then flyctl:"
        echo "   /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
        echo "   brew install flyctl"
        echo ""
        echo "5. Manual download from GitHub:"
        echo "   Visit: https://github.com/superfly/flyctl/releases"
        echo ""
        echo "For Windows users:"
        echo "   PowerShell: iwr https://fly.io/install.ps1 -useb | iex"
        echo ""
        print_status "After installation, you may need to restart your terminal or run:"
        echo "   source ~/.bashrc  # or ~/.zshrc"
        echo ""
        print_status "See DEPLOYMENT.md for detailed installation instructions"
        exit 1
    fi
    
    # Verify flyctl is working
    local flyctl_version
    if flyctl_version=$(flyctl version 2>/dev/null); then
        print_success "flyctl is installed: $(echo "$flyctl_version" | head -1)"
    else
        print_error "flyctl is installed but not working properly"
        echo "Try reinstalling flyctl using: ./install-flyctl.sh"
        exit 1
    fi
    
    # Check if logged in
    if ! flyctl auth whoami &> /dev/null; then
        print_error "Not logged in to Fly.io"
        echo ""
        echo "Please login to Fly.io first:"
        echo "   flyctl auth login"
        echo ""
        echo "This will open your browser to authenticate with Fly.io"
        exit 1
    fi
    
    local whoami_output
    if whoami_output=$(flyctl auth whoami 2>/dev/null); then
        print_success "Logged in to Fly.io as: $whoami_output"
    else
        print_warning "Logged in to Fly.io (unable to determine username)"
    fi
}

# Check required files
check_files() {
    print_header "Checking Required Files"
    
    required_files=("fly.toml" "Dockerfile" "main.go" ".env")
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            print_error "Required file missing: $file"
            exit 1
        fi
        print_success "Found: $file"
    done
}

# Read environment variables from .env file
read_env_file() {
    print_header "Reading Environment Variables"
    
    if [[ ! -f ".env" ]]; then
        print_error ".env file not found"
        exit 1
    fi
    
    # Source the .env file
    set -a  # automatically export all variables
    source .env
    set +a  # stop automatically exporting
    
    # Check required variables
    if [[ -z "$CUSTOMERIO_SITE_ID" ]]; then
        print_error "CUSTOMERIO_SITE_ID not found in .env file"
        exit 1
    fi
    
    if [[ -z "$CUSTOMERIO_API_KEY" ]]; then
        print_error "CUSTOMERIO_API_KEY not found in .env file"
        exit 1
    fi
    
    print_success "Environment variables loaded from .env"
    print_status "CUSTOMERIO_SITE_ID: ${CUSTOMERIO_SITE_ID}"
    print_status "CUSTOMERIO_API_KEY: ${CUSTOMERIO_API_KEY:0:10}... (truncated)"
}

# Create Fly.io app if it doesn't exist
create_app() {
    print_header "Creating Fly.io App"
    
    # Check if app already exists
    if flyctl apps list | grep -q "customerio-pauser"; then
        print_success "App 'customerio-pauser' already exists"
        return 0
    fi
    
    print_status "Creating new Fly.io app 'customerio-pauser'..."
    
    # Create the app in the specified region
    if flyctl apps create customerio-pauser --org personal; then
        print_success "App 'customerio-pauser' created successfully"
    else
        print_error "Failed to create app 'customerio-pauser'"
        echo ""
        echo "This might be because:"
        echo "  - The app name is already taken by another user"
        echo "  - You don't have permission to create apps"
        echo "  - Network connectivity issues"
        echo ""
        echo "You can try:"
        echo "  1. Choose a different app name in fly.toml"
        echo "  2. Check your Fly.io account permissions"
        echo "  3. Run 'flyctl auth whoami' to verify authentication"
        exit 1
    fi
}

# Set up Fly.io secrets
setup_secrets() {
    print_header "Setting Up Fly.io Secrets"
    
    print_status "Setting CUSTOMERIO_SITE_ID..."
    if flyctl secrets set CUSTOMERIO_SITE_ID="$CUSTOMERIO_SITE_ID" --stage; then
        print_success "CUSTOMERIO_SITE_ID set successfully"
    else
        print_error "Failed to set CUSTOMERIO_SITE_ID"
        exit 1
    fi
    
    print_status "Setting CUSTOMERIO_API_KEY..."
    if flyctl secrets set CUSTOMERIO_API_KEY="$CUSTOMERIO_API_KEY" --stage; then
        print_success "CUSTOMERIO_API_KEY set successfully"
    else
        print_error "Failed to set CUSTOMERIO_API_KEY"
        exit 1
    fi
    
    print_status "Listing current secrets..."
    flyctl secrets list
}

# Create required volumes
setup_volumes() {
    print_header "Setting Up Volumes"
    
    # Check if volume already exists
    if flyctl volumes list | grep -q "app_logs"; then
        print_success "Volume 'app_logs' already exists"
    else
        print_status "Creating volume 'app_logs'..."
        if flyctl volumes create app_logs --region iad --size 1; then
            print_success "Volume 'app_logs' created successfully"
        else
            print_error "Failed to create volume 'app_logs'"
            exit 1
        fi
    fi
}

# Deploy the application
deploy_app() {
    print_header "Deploying Application"
    
    print_status "Starting deployment to Fly.io..."
    
    if flyctl deploy; then
        print_success "Deployment completed successfully!"
    else
        print_error "Deployment failed"
        exit 1
    fi
}

# Post-deployment verification
verify_deployment() {
    print_header "Post-Deployment Verification"
    
    print_status "Checking application status..."
    flyctl status
    
    print_status "Checking application health..."
    sleep 10  # Give the app time to start
    
    # Get the app URL
    app_url=$(flyctl info | grep "Hostname" | awk '{print $2}' | head -1)
    
    if [[ -n "$app_url" ]]; then
        print_status "Testing ping endpoint..."
        if curl -f "https://$app_url/ping" &> /dev/null; then
            print_success "Ping endpoint is responding"
        else
            print_warning "Ping endpoint is not responding yet (may need more time to start)"
        fi
        
        print_success "Application deployed successfully!"
        echo ""
        echo "🚀 Your application is available at:"
        echo "   https://$app_url"
        echo ""
        echo "📊 Useful commands:"
        echo "   flyctl logs          - View live logs"
        echo "   flyctl status        - Check application status"
        echo "   flyctl ssh console   - SSH into the application"
        echo "   flyctl metrics       - View application metrics"
    else
        print_warning "Could not determine application URL"
    fi
}

# Show logs
show_logs() {
    print_header "Recent Application Logs"
    
    print_status "Showing recent logs (press Ctrl+C to exit)..."
    flyctl logs
}

# Main execution
main() {
    echo -e "${GREEN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                Customer.io Pauser Deployment                 ║"
    echo "║                     Fly.io Deployment Script                 ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    # Run all checks and deployment steps
    check_flyctl
    check_files
    read_env_file
    create_app
    setup_secrets
    setup_volumes
    deploy_app
    verify_deployment
    
    echo ""
    print_success "🎉 Deployment process completed successfully!"
    echo ""
    
    # Ask if user wants to see logs
    read -p "Would you like to view live logs? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        show_logs
    fi
}

# Handle script interruption
trap 'echo -e "\n${YELLOW}Deployment interrupted by user${NC}"; exit 1' INT

# Run main function
main "$@"