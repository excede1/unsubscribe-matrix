#!/bin/bash

# Customer.io Pauser - Local Testing Script
# This script builds and tests the application locally with different logging modes

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

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        echo "Please install Go from: https://golang.org/dl/"
        exit 1
    fi
    print_success "Go is installed: $(go version)"
    
    # Check if Docker is installed (optional)
    if command -v docker &> /dev/null; then
        print_success "Docker is available: $(docker --version)"
        DOCKER_AVAILABLE=true
    else
        print_warning "Docker is not available - skipping Docker tests"
        DOCKER_AVAILABLE=false
    fi
}

# Check required files
check_files() {
    print_header "Checking Required Files"
    
    required_files=("main.go" "go.mod" ".env" "views/index.html")
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            print_error "Required file missing: $file"
            exit 1
        fi
        print_success "Found: $file"
    done
}

# Load environment variables
load_env() {
    print_header "Loading Environment Variables"
    
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
    
    print_success "Environment variables loaded"
    print_status "CUSTOMERIO_SITE_ID: ${CUSTOMERIO_SITE_ID}"
    print_status "CUSTOMERIO_API_KEY: ${CUSTOMERIO_API_KEY:0:10}... (truncated)"
}

# Build the application
build_app() {
    print_header "Building Application"
    
    print_status "Installing Go dependencies..."
    if go mod tidy; then
        print_success "Dependencies updated"
    else
        print_error "Failed to update dependencies"
        exit 1
    fi
    
    print_status "Building Go application..."
    if go build -o customerio-pauser-test main.go; then
        print_success "Application built successfully"
    else
        print_error "Failed to build application"
        exit 1
    fi
}

# Test with file logging (default development mode)
test_file_logging() {
    print_header "Testing File Logging Mode"
    
    print_status "Starting application with file logging..."
    print_status "Log output will be written to app.log"
    print_status "Application will be available at http://localhost:3000"
    print_status "Press Ctrl+C to stop the application"
    
    # Remove existing log file
    rm -f app.log
    
    # Set environment for file logging
    export PORT=3000
    unset FLY_APP_NAME  # Ensure we're in development mode
    unset LOG_TO_FILE   # Use default file logging
    
    echo ""
    print_warning "Starting application in 3 seconds..."
    sleep 3
    
    # Start the application in background
    ./customerio-pauser-test &
    APP_PID=$!
    
    # Wait for application to start
    sleep 5
    
    # Test the application
    print_status "Testing application endpoints..."
    
    # Test ping endpoint
    if curl -f http://localhost:3000/ping &> /dev/null; then
        print_success "Ping endpoint is responding"
    else
        print_error "Ping endpoint is not responding"
    fi
    
    # Test main page
    if curl -f http://localhost:3000/ &> /dev/null; then
        print_success "Main page is responding"
    else
        print_error "Main page is not responding"
    fi
    
    # Show log file contents
    if [[ -f "app.log" ]]; then
        print_success "Log file created successfully"
        print_status "Recent log entries:"
        echo "----------------------------------------"
        tail -10 app.log
        echo "----------------------------------------"
    else
        print_error "Log file was not created"
    fi
    
    # Stop the application
    print_status "Stopping application..."
    kill $APP_PID 2>/dev/null || true
    wait $APP_PID 2>/dev/null || true
    
    print_success "File logging test completed"
}

# Test with stdout logging
test_stdout_logging() {
    print_header "Testing Stdout Logging Mode"
    
    print_status "Starting application with stdout logging..."
    print_status "Log output will be displayed in terminal"
    print_status "Application will be available at http://localhost:3001"
    print_status "Press Ctrl+C to stop the application"
    
    # Set environment for stdout logging
    export PORT=3001
    export LOG_TO_FILE=false
    unset FLY_APP_NAME  # Ensure we're in development mode
    
    echo ""
    print_warning "Starting application in 3 seconds..."
    sleep 3
    
    # Start the application in background
    ./customerio-pauser-test &
    APP_PID=$!
    
    # Wait for application to start
    sleep 5
    
    # Test the application
    print_status "Testing application endpoints..."
    
    # Test ping endpoint
    if curl -f http://localhost:3001/ping &> /dev/null; then
        print_success "Ping endpoint is responding"
    else
        print_error "Ping endpoint is not responding"
    fi
    
    # Test main page
    if curl -f http://localhost:3001/ &> /dev/null; then
        print_error "Main page is responding"
    else
        print_error "Main page is not responding"
    fi
    
    # Stop the application
    print_status "Stopping application..."
    kill $APP_PID 2>/dev/null || true
    wait $APP_PID 2>/dev/null || true
    
    print_success "Stdout logging test completed"
}

# Test environment detection
test_environment_detection() {
    print_header "Testing Environment Detection"
    
    print_status "Testing development environment detection..."
    unset FLY_APP_NAME
    export PORT=3002
    
    ./customerio-pauser-test &
    APP_PID=$!
    sleep 3
    
    if curl -f http://localhost:3002/ping &> /dev/null; then
        print_success "Development environment detected correctly"
    else
        print_error "Development environment detection failed"
    fi
    
    kill $APP_PID 2>/dev/null || true
    wait $APP_PID 2>/dev/null || true
    
    print_status "Testing production environment simulation..."
    export FLY_APP_NAME=customerio-pauser
    export PORT=3003
    
    ./customerio-pauser-test &
    APP_PID=$!
    sleep 3
    
    if curl -f http://localhost:3003/ping &> /dev/null; then
        print_success "Production environment simulation works"
    else
        print_error "Production environment simulation failed"
    fi
    
    kill $APP_PID 2>/dev/null || true
    wait $APP_PID 2>/dev/null || true
    
    unset FLY_APP_NAME  # Reset for other tests
}

# Test Docker build (if Docker is available)
test_docker_build() {
    if [[ "$DOCKER_AVAILABLE" != true ]]; then
        print_warning "Skipping Docker tests - Docker not available"
        return
    fi
    
    print_header "Testing Docker Build"
    
    print_status "Building Docker image..."
    if docker build -t customerio-pauser-local-test .; then
        print_success "Docker image built successfully"
    else
        print_error "Docker build failed"
        return
    fi
    
    print_status "Testing Docker container..."
    
    # Create a temporary env file for Docker
    cat > .env.docker << EOF
CUSTOMERIO_SITE_ID=${CUSTOMERIO_SITE_ID}
CUSTOMERIO_API_KEY=${CUSTOMERIO_API_KEY}
PORT=3000
EOF
    
    # Run container in background
    docker run -d --name customerio-pauser-test -p 3004:3000 --env-file .env.docker customerio-pauser-local-test
    
    # Wait for container to start
    sleep 10
    
    # Test the containerized application
    if curl -f http://localhost:3004/ping &> /dev/null; then
        print_success "Docker container is responding"
    else
        print_error "Docker container is not responding"
    fi
    
    # Show container logs
    print_status "Container logs:"
    echo "----------------------------------------"
    docker logs customerio-pauser-test | tail -10
    echo "----------------------------------------"
    
    # Clean up
    docker stop customerio-pauser-test &> /dev/null || true
    docker rm customerio-pauser-test &> /dev/null || true
    docker rmi customerio-pauser-local-test &> /dev/null || true
    rm -f .env.docker
    
    print_success "Docker test completed"
}

# Interactive test menu
interactive_menu() {
    while true; do
        echo ""
        print_header "Local Testing Menu"
        echo "1. Test File Logging Mode"
        echo "2. Test Stdout Logging Mode"
        echo "3. Test Environment Detection"
        echo "4. Test Docker Build"
        echo "5. Run All Tests"
        echo "6. Exit"
        echo ""
        read -p "Select an option (1-6): " choice
        
        case $choice in
            1)
                test_file_logging
                ;;
            2)
                test_stdout_logging
                ;;
            3)
                test_environment_detection
                ;;
            4)
                test_docker_build
                ;;
            5)
                test_file_logging
                test_stdout_logging
                test_environment_detection
                test_docker_build
                ;;
            6)
                print_success "Exiting..."
                break
                ;;
            *)
                print_error "Invalid option. Please select 1-6."
                ;;
        esac
    done
}

# Cleanup function
cleanup() {
    print_status "Cleaning up..."
    
    # Kill any running test processes
    pkill -f customerio-pauser-test 2>/dev/null || true
    
    # Remove test binary
    rm -f customerio-pauser-test
    
    # Remove any Docker containers
    if [[ "$DOCKER_AVAILABLE" == true ]]; then
        docker stop customerio-pauser-test &> /dev/null || true
        docker rm customerio-pauser-test &> /dev/null || true
    fi
    
    print_success "Cleanup completed"
}

# Main execution
main() {
    echo -e "${GREEN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                Customer.io Pauser Local Testing              ║"
    echo "║                     Environment & Logging Tests              ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    # Run prerequisite checks
    check_prerequisites
    check_files
    load_env
    build_app
    
    # Check if running in interactive mode
    if [[ "$1" == "--interactive" ]] || [[ "$1" == "-i" ]]; then
        interactive_menu
    else
        # Run all tests
        print_header "Running All Tests"
        test_file_logging
        test_stdout_logging
        test_environment_detection
        test_docker_build
        
        print_success "🎉 All tests completed!"
    fi
}

# Handle script interruption
trap 'echo -e "\n${YELLOW}Testing interrupted by user${NC}"; cleanup; exit 1' INT

# Handle script exit
trap 'cleanup' EXIT

# Run main function
main "$@"