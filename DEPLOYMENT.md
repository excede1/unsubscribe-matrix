# Customer.io Pauser - Fly.io Deployment Guide

This guide provides comprehensive instructions for deploying the Customer.io Pauser application to Fly.io.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Initial Setup](#initial-setup)
3. [Secrets Management](#secrets-management)
4. [Deployment Process](#deployment-process)
5. [Environment Detection](#environment-detection)
6. [Monitoring and Logging](#monitoring-and-logging)
7. [Updates and Maintenance](#updates-and-maintenance)
8. [Troubleshooting](#troubleshooting)

## Prerequisites

### 1. Install Fly.io CLI

The Fly.io CLI (`flyctl`) is required for deployment. Choose the installation method that works best for your setup:

#### Option 1: Direct Download (Recommended - No Homebrew Required)

**macOS/Linux:**
```bash
curl -L https://fly.io/install.sh | sh
```

This method:
- Works on both Intel and Apple Silicon Macs
- Doesn't require Homebrew or other package managers
- Automatically adds flyctl to your PATH
- Is the fastest way to get started

**After installation, you may need to add flyctl to your PATH:**
```bash
# Add to your shell profile (.bashrc, .zshrc, etc.)
export PATH="$HOME/.fly/bin:$PATH"

# Or reload your shell
source ~/.bashrc  # or ~/.zshrc
```

#### Option 2: Using Homebrew (If you have Homebrew installed)

**macOS:**
```bash
brew install flyctl
```

**If you don't have Homebrew and want to install it first:**
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
brew install flyctl
```

#### Option 3: Manual Download from GitHub

1. Visit the [Fly.io CLI releases page](https://github.com/superfly/flyctl/releases)
2. Download the appropriate binary for your system:
   - `flyctl_*_macOS_x86_64.tar.gz` (Intel Mac)
   - `flyctl_*_macOS_arm64.tar.gz` (Apple Silicon Mac)
   - `flyctl_*_Linux_x86_64.tar.gz` (Linux)
3. Extract and move to a directory in your PATH:
   ```bash
   tar -xzf flyctl_*.tar.gz
   sudo mv flyctl /usr/local/bin/
   chmod +x /usr/local/bin/flyctl
   ```

#### Option 4: Alternative Package Managers

**Using MacPorts (macOS):**
```bash
sudo port install flyctl
```

**Using Nix:**
```bash
nix-env -iA nixpkgs.flyctl
```

#### Option 5: Windows Installation

**PowerShell:**
```powershell
iwr https://fly.io/install.ps1 -useb | iex
```

**Using Chocolatey:**
```powershell
choco install flyctl
```

**Using Scoop:**
```powershell
scoop install flyctl
```

#### Automated Installation Script

For convenience, you can use the provided installation script that automatically detects your system and chooses the best installation method:

```bash
./install-flyctl.sh
```

This script will:
- Check if flyctl is already installed
- Detect if Homebrew is available
- Use the most appropriate installation method for your system
- Verify the installation was successful
- Provide next steps

### 2. Verify Installation
```bash
flyctl version
```

You should see output similar to:
```
flyctl v0.x.x darwin/arm64 Commit: abc123 BuildDate: 2024-01-01T00:00:00Z
```

### 3. Login to Fly.io
```bash
flyctl auth login
```

### 4. Required Files
Ensure these files exist in your project directory:
- `fly.toml` - Fly.io configuration
- `Dockerfile` - Container build instructions
- `.dockerignore` - Build optimization
- `main.go` - Application source code
- `.env` - Local environment variables (for reference)

## Initial Setup

### 1. Create Fly.io Application
If you haven't already created the application:
```bash
flyctl apps create customerio-pauser
```

### 2. Create Required Volume
The application uses a persistent volume for logging:
```bash
flyctl volumes create app_logs --region iad --size 1
```

## Secrets Management

The application requires Customer.io API credentials to be set as Fly.io secrets.

### Manual Secret Setup
```bash
# Set Customer.io Site ID
flyctl secrets set CUSTOMERIO_SITE_ID=your_site_id_here

# Set Customer.io API Key
flyctl secrets set CUSTOMERIO_API_KEY=your_api_key_here
```

### Automated Secret Setup (using deploy.sh)
The included `deploy.sh` script can automatically read your `.env` file and set up secrets:
```bash
./deploy.sh
```

### Verify Secrets
```bash
flyctl secrets list
```

## Deployment Process

### Option 1: Automated Deployment (Recommended)

**If flyctl is not installed, install it first:**
```bash
# Use the automated installer
chmod +x install-flyctl.sh
./install-flyctl.sh
```

**Then deploy using the deployment script:**
```bash
chmod +x deploy.sh
./deploy.sh
```

The deployment script will automatically:
- Check if flyctl is installed and provide installation instructions if not
- Verify you're logged in to Fly.io
- Read environment variables from `.env` file
- Set up Fly.io secrets
- Create required volumes
- Deploy the application
- Verify the deployment

### Option 2: Manual Deployment

**Prerequisites check:**
```bash
# Ensure flyctl is installed
flyctl version

# Ensure you're logged in
flyctl auth login
```

**Deploy manually:**
```bash
# Deploy the application
flyctl deploy

# Check deployment status
flyctl status

# View logs
flyctl logs
```

### Troubleshooting Installation Issues

If you encounter the error `bash: brew: command not found`, it means Homebrew is not installed. You have several options:

1. **Use direct download (fastest):**
   ```bash
   curl -L https://fly.io/install.sh | sh
   ```

2. **Use the automated installer:**
   ```bash
   ./install-flyctl.sh
   ```

3. **Install Homebrew first, then flyctl:**
   ```bash
   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
   brew install flyctl
   ```

The deployment script ([`deploy.sh`](deploy.sh)) will detect if flyctl is missing and provide detailed installation instructions.

## Environment Detection

The application automatically detects its environment:

### Production Environment (Fly.io)
- **Detection**: Presence of `FLY_APP_NAME` environment variable
- **Logging**: Outputs to stdout for Fly.io log aggregation
- **Features**: 
  - Skips `.env` file loading
  - Disables port killing functionality
  - Uses production-optimized settings

### Development Environment (Local)
- **Detection**: Absence of `FLY_APP_NAME` environment variable
- **Logging**: Outputs to `app.log` file by default
- **Features**:
  - Loads `.env` file for configuration
  - Kills existing processes on port before starting
  - Supports `LOG_TO_FILE=false` for stdout logging

## Monitoring and Logging

### View Live Logs
```bash
flyctl logs
```

### View Application Status
```bash
flyctl status
```

### Health Check Endpoints
- **Ping**: `https://customerio-pauser.fly.dev/ping`
- **Main Interface**: `https://customerio-pauser.fly.dev/`

### Application Metrics
```bash
flyctl metrics
```

### SSH into Running Instance
```bash
flyctl ssh console
```

## Updates and Maintenance

### Deploying Updates
1. Make your code changes
2. Run the deployment script:
   ```bash
   ./deploy.sh
   ```
3. Monitor the deployment:
   ```bash
   flyctl logs
   ```

### Scaling
```bash
# Scale to multiple instances
flyctl scale count 2

# Scale memory/CPU
flyctl scale memory 512
```

### Restart Application
```bash
flyctl apps restart customerio-pauser
```

## Troubleshooting

### Common Issues

#### 0. flyctl Installation Issues

**Symptoms**: `bash: brew: command not found` or `flyctl: command not found`

**Solutions**:

**For "brew: command not found":**
This means Homebrew is not installed on your system. You have several options:

1. **Use direct download (recommended - no Homebrew required):**
   ```bash
   curl -L https://fly.io/install.sh | sh
   ```

2. **Use the automated installer:**
   ```bash
   ./install-flyctl.sh
   ```

3. **Install Homebrew first, then flyctl:**
   ```bash
   # Install Homebrew
   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
   
   # Then install flyctl
   brew install flyctl
   ```

4. **Manual download from GitHub:**
   - Visit: https://github.com/superfly/flyctl/releases
   - Download the appropriate binary for your system
   - Extract and move to `/usr/local/bin/`

**For "flyctl: command not found" after installation:**
```bash
# Add flyctl to your PATH
export PATH="$HOME/.fly/bin:$PATH"

# Make it permanent by adding to your shell profile
echo 'export PATH="$HOME/.fly/bin:$PATH"' >> ~/.bashrc  # or ~/.zshrc

# Reload your shell
source ~/.bashrc  # or ~/.zshrc
```

**Verify installation:**
```bash
flyctl version
flyctl auth login
```

#### 1. Application Won't Start
**Symptoms**: Application crashes on startup
**Solutions**:
- Check secrets are set: `flyctl secrets list`
- Verify logs: `flyctl logs`
- Ensure Customer.io credentials are valid

#### 2. Secrets Not Found
**Symptoms**: "CUSTOMERIO_SITE_ID not set" or "CUSTOMERIO_API_KEY not set"
**Solutions**:
```bash
# List current secrets
flyctl secrets list

# Set missing secrets
flyctl secrets set CUSTOMERIO_SITE_ID=your_site_id
flyctl secrets set CUSTOMERIO_API_KEY=your_api_key

# Restart application
flyctl apps restart customerio-pauser
```

#### 3. Build Failures
**Symptoms**: Docker build fails during deployment
**Solutions**:
- Check Dockerfile syntax
- Verify all required files are present
- Check `.dockerignore` isn't excluding necessary files
- Try local build: `docker build -t test .`

#### 4. Health Check Failures
**Symptoms**: Application shows as unhealthy
**Solutions**:
- Test ping endpoint: `curl https://customerio-pauser.fly.dev/ping`
- Check application logs: `flyctl logs`
- Verify port 3000 is properly exposed

#### 5. Volume Issues
**Symptoms**: Logging errors or volume mount failures
**Solutions**:
```bash
# List volumes
flyctl volumes list

# Create volume if missing
flyctl volumes create app_logs --region iad --size 1

# Check volume status
flyctl volumes show app_logs
```

### Debug Commands

```bash
# View detailed application information
flyctl info

# Check machine status
flyctl machine list

# View configuration
flyctl config show

# Test local build
docker build -t customerio-pauser-test .
docker run -p 3000:3000 --env-file .env customerio-pauser-test
```

### Getting Help

1. **Fly.io Documentation**: https://fly.io/docs/
2. **Fly.io Community**: https://community.fly.io/
3. **Application Logs**: `flyctl logs` for real-time debugging
4. **Local Testing**: Use `local-test.sh` to test changes locally first

### Configuration Reference

#### fly.toml Key Settings
- **Region**: `iad` (US East - Virginia)
- **Port**: `3000` (internal), `80/443` (external)
- **Memory**: `256MB`
- **Auto-scaling**: Disabled (always running)
- **Health checks**: `/ping` endpoint
- **Volume**: `app_logs` mounted at `/app/logs`

#### Environment Variables
- `PORT`: Set to `3000` in fly.toml
- `FLY_APP_NAME`: Automatically set by Fly.io (used for environment detection)
- `CUSTOMERIO_SITE_ID`: Set via secrets
- `CUSTOMERIO_API_KEY`: Set via secrets

## Security Notes

1. **Never commit secrets**: Keep `.env` file in `.gitignore`
2. **Use Fly.io secrets**: Always use `flyctl secrets set` for sensitive data
3. **HTTPS only**: Application forces HTTPS in production
4. **Basic Auth**: Customer.io API uses Site ID/API Key as Basic Auth credentials

## Performance Optimization

1. **Multi-stage Docker build**: Reduces final image size
2. **Minimal base image**: Uses `alpine` for smaller footprint
3. **Health checks**: Ensures application availability
4. **Persistent logging**: Volume-mounted logs for debugging
5. **Connection limits**: Configured for optimal performance

---

For additional support or questions, refer to the troubleshooting section or check the application logs using `flyctl logs`.