# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SwimMate is a web application for tracking swim training sessions. It's built with Go using a server-rendered HTML
architecture with PostgreSQL for data persistence.

**Technology Stack:**

- Backend: Go 1.25
- Web Framework: `httprouter` for routing, `alice` for middleware chaining
- Database: PostgreSQL with `lib/pq` driver
- Session Management: `scs/v2` with PostgreSQL store
- Templates: Go's `html/template` with embedded filesystem
- Authentication: bcrypt password hashing

## Development Commands

### Local Development

```bash
# Start the application with hot reload using Air
docker-compose up

# The application will be available at http://localhost:8998
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/models

# Run static analysis
go vet ./...
```

### Linting

```bash
golangci-lint run
```

### Database Seeding

Generate test users with random swim data using the seed command:

```bash
# Create a test user with 50 swims over the past year
DB_DSN="postgres://swimmate:swimmate@localhost:5432/swimmate?sslmode=disable" \
go run ./cmd/seed -username testuser

# Create user with custom options
DB_DSN="postgres://swimmate:swimmate@localhost:5432/swimmate?sslmode=disable" \
go run ./cmd/seed \
  -username john_swimmer \
  -password mysecretpass \
  -first-name John \
  -last-name Doe \
  -swims 100 \
  -days-back 730
```

Available flags: `-username` (required), `-password`, `-first-name`, `-last-name`, `-email`, `-swims`, `-days-back`. See
`cmd/seed/README.md` for details.

## Architecture

### Application Structure

**Standard Go Layout:**

- `cmd/web/` - Main application entry point and HTTP handlers
- `cmd/seed/` - Database seeding tool for creating test users with swim data
- `internal/models/` - Data models and database interactions
- `ui/` - Embedded filesystem containing HTML templates and static assets
- `remote/production/` - Production deployment configuration (Caddy, systemd)

**Key Components:**

1. **Application Context** (`cmd/web/main.go`):
    - Central `application` struct holds logger, database models, template cache, session manager, and version
    - Server runs on port 8998
    - Database connection string from `DB_DSN` environment variable

2. **Routing** (`cmd/web/routes.go`):
    - Uses `httprouter` for routing
    - Three middleware chains:
        - `standard`: All requests (panic recovery, logging, security headers)
        - `dynamic`: Session management
        - `protected`: Requires authentication
    - Static files served from embedded `ui.Files` filesystem

3. **Models** (`internal/models/`):
    - Interface-based design for testability
    - `SwimModel`: CRUD operations for swim records, pagination, and summary statistics
    - `UserModel`: Authentication only (no registration)
    - Both use raw SQL queries (no ORM)

4. **Templates** (`ui/html/`):
    - Base template: `base.tmpl`
    - Page templates: `html/pages/*.tmpl`
    - Partials: `html/partials/*.tmpl`
    - Custom template functions: `numberFormat`, `sub`, `add`, `seq`, `min`, `emptyStars`
    - Templates compiled into binary via embed

5. **Middleware** (`cmd/web/middleware.go`):
    - `secureHeaders`: Sets security headers (CSP, X-Frame-Options, etc.)
    - `logRequest`: Logs all incoming requests with slog
    - `recoverPanic`: Panic recovery
    - `requireAuthentication`: Redirects to login if not authenticated

### Data Flow

1. **Authentication Flow:**
    - Login form at `/login`
    - POST to `/authenticate` validates credentials against PostgreSQL
    - Session stored in PostgreSQL via `scs` session manager
    - Protected routes check `authenticatedUserID` in session

2. **Swim Tracking:**
    - Home page (`/`) displays swim summary statistics (total, monthly, weekly)
    - `/swims` shows paginated list of swims (20 per page)
    - `/swims/more` for AJAX-style pagination
    - `/yearly-figures` shows year/month breakdown
    - `/swim` GET/POST for creating new swim entries

3. **Database Schema:**
    - `swims` table: id, date, distance_m, assessment, user_id
    - `users` table: id, first_name, last_name, username, email, password, date_joined, last_login
    - Session data stored via `scs` in dedicated session table

### Template Data Structure

All templates receive a `templateData` struct:

- `Version`: App version (from build ldflags or "development")
- `Data`: Page-specific data (typically model data)
- `Flash`: Flash messages with type (success/error)
- `IsAuthenticated`: Boolean for auth status
- `CurrentDate`: Today's date in ISO format

## CI/CD Pipeline

The GitHub Actions workflow (`.github/workflows/cicd.yml`) runs on pushes to main:

1. **Test**: Runs `go test ./...` and `go vet ./...`
2. **Lint**: Runs `golangci-lint`
3. **Build**: Verifies `go build ./...`
4. **Versioning**: Auto-increments patch version
5. **Release**: Creates GitHub release with notes
6. **Deploy**: Builds Linux binary, SCPs to server, restarts systemd service

## Database Connection

- Uses PostgreSQL connection pooling via `database/sql`
- Connection string from `DB_DSN` environment variable
- Format: `postgres://user:pass@host:port/dbname?sslmode=disable`
- Local dev via docker-compose: `swimmate` user/pass/db on port 5432

## Important Notes

- No user registration functionality - users must be created directly in database
- Passwords are bcrypt hashed (use appropriate cost factor)
- Sessions expire after 12 hours
- Application version injected at build time via ldflags: `-X main.version=<version>`
- All templates must be in `ui/html/` to be embedded in binary
- Static assets served from `/static/` route map to `ui/static/` directory

## Version control

Keep commits atomic: commit only the files you touched and list each path explicitly.
For tracked files run `git commit -m "<scoped message>" -- path/to/file1 path/to/file2`.
For brand-new files, use the one-liner
`git restore --staged :/ && git add "path/to/file1" "path/to/file2" && git commit -m "<scoped message>" -- path/to/file1 path/to/file2`

**Commit Message Guidelines:**
- Subject line must be 50 characters or less
- Use imperative mood (e.g., "Fix bug" not "Fixed bug")
- Keep commits atomic and focused on a single change
