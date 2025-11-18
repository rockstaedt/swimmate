# Seed Command

This automation tool creates a test user with random swim entries for development and testing purposes.

## Usage

### Basic Usage

Create a user with default settings (50 swims over the past year):

```bash
DB_DSN="postgres://swimmate:swimmate@localhost:5432/swimmate?sslmode=disable" \
go run ./cmd/seed -username testuser
```

### Custom Options

```bash
DB_DSN="postgres://swimmate:swimmate@localhost:5432/swimmate?sslmode=disable" \
go run ./cmd/seed \
  -username john_swimmer \
  -password mysecretpass \
  -first-name John \
  -last-name Doe \
  -email john@example.com \
  -swims 100 \
  -days-back 730
```

### Available Flags

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `-username` | Username for the new user | - | **Yes** |
| `-password` | Password for the new user | `password123` | No |
| `-first-name` | First name | `Test` | No |
| `-last-name` | Last name | `User` | No |
| `-email` | Email address | `{username}@example.com` | No |
| `-swims` | Number of swim entries to create | `50` | No |
| `-days-back` | Generate swims going back this many days | `365` | No |

## Examples

### Quick test user
```bash
DB_DSN="postgres://swimmate:swimmate@localhost:5432/swimmate?sslmode=disable" \
go run ./cmd/seed -username demo
```
Creates user `demo` with password `password123` and 50 swim entries over the past year.

### Heavy testing data
```bash
DB_DSN="postgres://swimmate:swimmate@localhost:5432/swimmate?sslmode=disable" \
go run ./cmd/seed -username poweruser -swims 500 -days-back 1095
```
Creates user `poweruser` with 500 swim entries spanning 3 years.

### Using with Docker Compose

If your database is running via docker-compose:

```bash
DB_DSN="postgres://swimmate:swimmate@localhost:5432/swimmate?sslmode=disable" \
go run ./cmd/seed -username testuser
```

## Generated Data

- **Swim distances**: Random realistic distances (500m, 750m, 1000m, 1200m, 1500m, 1800m, 2000m, 2500m, 3000m, 3500m, 4000m)
- **Swim dates**: Randomly distributed over the specified time period (days-back)
- **Assessments**: Random rating from 0 to 2 stars
- **Password**: Bcrypt hashed with default cost

## Notes

- The tool will fail if a user with the same username already exists
- Make sure the database is running and accessible before running the seed command
- The DB_DSN environment variable must be set
