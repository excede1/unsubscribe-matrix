# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Development
- **Run application**: `go run main.go` or build first with `go build -o main . && ./main`
- **Build**: `go build -o main .`
- **Test**: `go test ./...`
- **Format**: `go fmt ./...`
- **Lint**: `golangci-lint run` (if available)
- **Dependencies**: `go mod download`
- **Update dependencies**: `go mod tidy`

### Testing & Verification
- **Local test script**: `./local-test.sh` - comprehensive testing of all endpoints
- **Verify configuration**: `./verify-config.sh` - check environment setup
- **Health check**: `curl http://localhost:3000/ping`

### Deployment
- **Deploy to Fly.io**: `./deploy.sh` or `flyctl deploy`
- **Set secrets**: `flyctl secrets set KEY=value`
- **View logs**: `flyctl logs`

### Database
- **SQLite database file**: `email_processing.db`
- **Backup**: `cp email_processing.db email_processing_backup_$(date +%Y%m%d).db`

## Architecture

### Technology Stack
- **Language**: Go 1.23
- **Web Framework**: Fiber v2 (high-performance HTTP framework)
- **Database**: SQLite (embedded database)
- **Template Engine**: Fiber HTML template engine
- **API Integration**: Customer.io Track API

### Project Structure
```
├── main.go              # Main application logic, HTTP handlers, Customer.io API integration
├── database.go          # SQLite database operations and record management
├── views/              
│   ├── index.html      # Customer email preference interface
│   └── results.html    # Admin dashboard
├── assets/             # Static assets (logo)
└── *.sh                # Deployment and utility scripts
```

### Key Components

#### Customer.io Integration
- Uses Track API for managing customer attributes and relationships
- Authentication via Site ID and API Key (Base64 encoded)
- Three main operations:
  1. **Pause/Unpause**: Sets `paused` attribute on customer profile
  2. **International List**: Manages entity relationships (BBUS → BBAU)
  3. **Unsubscribe**: Sets `unsubscribed` attribute permanently

#### Database Schema
- Single table: `email_processing_records`
- Columns: `id` (INTEGER PRIMARY KEY), `timestamp` (DATETIME), `email` (TEXT), `action` (TEXT)
- Actions tracked: "PAUSE", "BBAU", "UNSUBSCRIBE"

#### Authentication
- Admin dashboard protected by HTTP Basic Auth
- Credentials from environment variables: `ADMIN_USERNAME`, `ADMIN_PASSWORD`

### Environment Variables
Required in `.env` file:
```
CUSTOMERIO_SITE_ID=     # Customer.io Site ID
CUSTOMERIO_API_KEY=     # Customer.io API Key
ADMIN_USERNAME=         # Admin dashboard username
ADMIN_PASSWORD=         # Admin dashboard password
PORT=                   # Server port (default: 3000)
```

### Endpoints
- `GET /` - Customer preference interface (requires `?email=` parameter)
- `GET /ping` - Health check
- `GET /results` - Admin dashboard (requires authentication)
- `GET /results/csv/:action` - Download CSV for specific action
- `POST /results/clear` - Clear all database records

### Error Handling
- All Customer.io API calls include comprehensive error logging
- Database operations wrapped in error handlers
- Failed operations logged to `app.log` (development) or stdout (production)

### Deployment
- Production detected via `FLY_APP_NAME` environment variable
- Fly.io configuration in `fly.toml`
- Docker support via `Dockerfile`
- Automated deployment via `deploy.sh` script