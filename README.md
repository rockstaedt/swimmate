# SwimMate

[![build](https://github.com/rockstaedt/swimmate/actions/workflows/cicd.yml/badge.svg)](https://github.com/rockstaedt/swimmate/actions/workflows/cicd.yml)
[![Latest tag](https://img.shields.io/github/v/tag/rockstaedt/swimmate)](https://github.com/rockstaedt/swimmate/releases)

SwimMate is a Go web application for tracking swim training sessions. It focuses on a fast, server-rendered HTML
experience and PostgreSQL for reliable persistence. The project is still under active development and breaking changes
are expected.

## Features

- Track every swim with date, distance, assessment, and owning user
- Summaries for total, monthly, and weekly volume on the dashboard
- Paginated swim history with `/swims/more` endpoint for AJAX-style loading
- Yearly breakdown charts for spotting progress across months
- Authenticated workflow with session-backed login

## Technology Stack

- **Language:** Go 1.25
- **Web:** `httprouter` for routing with `alice` middleware chains
- **Database:** PostgreSQL via `lib/pq`, accessed with raw SQL
- **Sessions:** `scs/v2` with PostgreSQL store
- **Templates:** Go `html/template` rendered from an embedded filesystem
- **Auth:** Bcrypt password hashing

## Repository Layout

```
cmd/web        # Main HTTP server, routes, middleware, templates wiring
cmd/seed       # CLI for generating demo users/swims
internal/models# Swim and User models plus DB helpers
ui             # HTML templates, partials, and static assets (embedded)
remote         # Production deployment scripts (Caddy, systemd, etc.)
```

## Getting Started

### Prerequisites

- Go 1.25 or newer
- Docker & Docker Compose (recommended for local Postgres)
- Make sure `golangci-lint` is installed if you plan to lint locally

### Quick Start (Docker)

```bash
docker-compose up
```

The app becomes available at http://localhost:8998. PostgreSQL is provided by the compose stack and preconfigured for
local development.

### Running the Server Manually

Set the `DB_DSN` to your PostgreSQL instance (example for the compose stack):

```bash
export DB_DSN="postgres://swimmate:swimmate@localhost:5432/swimmate?sslmode=disable"
go run ./cmd/web
```

The binary injects the application version through `-ldflags "-X main.version=<version>"` when building for production,
otherwise it defaults to `development`.

## Database Seeding

A helper CLI located in `cmd/seed` generates users with random swims:

```bash
DB_DSN="postgres://swimmate:swimmate@localhost:5432/swimmate?sslmode=disable" \
go run ./cmd/seed -username testuser
```

Additional flags:

- `-password`
- `-first-name`
- `-last-name`
- `-email`
- `-swims` (number of swim entries to create)
- `-days-back` (spread entries across the last N days)

See `cmd/seed/README.md` for complete details.

## Testing & Linting

```bash
# Run unit tests
go test ./...

# Static analysis
go vet ./...

# Linting (requires golangci-lint)
golangci-lint run
```

CI (see `.github/workflows/cicd.yml`) runs the same checks plus build verification, version bumping, release creation,
and deployment to the production host.

## Configuration

- `DB_DSN`: PostgreSQL connection string (required). Example:
  `postgres://swimmate:swimmate@localhost:5432/swimmate?sslmode=disable`
- Sessions expire after 12 hours and are stored in PostgreSQL via `scs/v2`
- Static assets are served from `/static/` mapped to `ui/static`

## Contributing

1. Fork or create a branch off `main`
2. Make focused changes (keep commits small and scoped)
3. Run `go test ./...`, `go vet ./...`, and `golangci-lint run`
4. Open a pull request describing the change and any relevant screenshots

Bug reports and feature requests are welcome via GitHub Issues.
