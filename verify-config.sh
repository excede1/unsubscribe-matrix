#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_status() { echo -e "${BLUE}[INFO]${NC} $1"; }

echo "🔍 Verifying Fly.io Configuration"
echo "=================================="

# Check fly.toml syntax
print_status "Checking fly.toml configuration..."
if flyctl config validate; then
    print_success "fly.toml configuration is valid"
else
    print_error "fly.toml configuration has errors"
    exit 1
fi

# Check if flyctl is working
print_status "Checking flyctl authentication..."
if flyctl auth whoami &> /dev/null; then
    whoami_output=$(flyctl auth whoami 2>/dev/null)
    print_success "Authenticated as: $whoami_output"
else
    print_error "Not authenticated with Fly.io"
    echo "Please run: flyctl auth login"
    exit 1
fi

# Check required files
print_status "Checking required files..."
required_files=("fly.toml" "Dockerfile" "main.go" ".env")
for file in "${required_files[@]}"; do
    if [[ -f "$file" ]]; then
        print_success "Found: $file"
    else
        print_error "Missing: $file"
        exit 1
    fi
done

# Check .env variables
print_status "Checking environment variables..."
if [[ -f ".env" ]]; then
    source .env
    if [[ -n "$CUSTOMERIO_SITE_ID" && -n "$CUSTOMERIO_API_KEY" ]]; then
        print_success "Environment variables are set"
        print_status "CUSTOMERIO_SITE_ID: ${CUSTOMERIO_SITE_ID}"
        print_status "CUSTOMERIO_API_KEY: ${CUSTOMERIO_API_KEY:0:10}... (truncated)"
    else
        print_error "Missing required environment variables in .env"
        exit 1
    fi
else
    print_error ".env file not found"
    exit 1
fi

echo ""
print_success "✅ All configuration checks passed!"
echo ""
print_status "You can now run the deployment with:"
echo "   ./deploy.sh"