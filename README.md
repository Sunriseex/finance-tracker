# Finance Tracker

Finance Tracker is a Go backend application for tracking personal finances. It stores accounts, transactions, transfers, interest rules, interest accruals, and migrated legacy deposit data in PostgreSQL.

The project is intended as a practical backend learning project with production-oriented practices: layered structure, PostgreSQL migrations, HTTP handlers, validation, CI, linting, integration tests, and API authentication.

## Features

* Account management: cash, card, savings, term deposit, broker, and other account types.
* Transaction tracking: income, expense, transfer, initial balance, and interest income transactions.
* Transfers between accounts.
* Account balance calculation.
* Interest rules for savings and deposits.
* Manual interest accrual with duplicate-accrual protection.
* PostgreSQL persistence.
* JSON legacy data migration.
* Protected HTTP API with Bearer token authentication.
* Health and readiness endpoints.
* CI with tests, race tests, linting, build checks, and migration checks.

## Tech Stack

* Go
* PostgreSQL
* Goose migrations
* Chi router
* golangci-lint
* GitHub Actions

## Project Structure

```text
.
├── cmd/
│   └── server/              # HTTP API entrypoint
├── configs/
│   └── example.env          # Example local configuration
├── internal/
│   ├── config/              # Application configuration
│   ├── http/
│   │   ├── dto/             # HTTP request/response DTOs
│   │   ├── handlers/        # HTTP handlers and routing
│   │   └── middleware/      # HTTP middleware
│   ├── migration/           # Legacy JSON migration logic
│   ├── models/              # Domain models
│   ├── postgres/            # PostgreSQL repositories/store
│   ├── repository/          # Repository interfaces/contracts
│   └── services/            # Business logic
├── migrations/              # PostgreSQL migrations
└── .github/workflows/       # CI configuration
```

## Requirements

* Go 1.26.2 or newer, based on the project `go.mod`.
* PostgreSQL 17 or compatible.
* `goose` for running migrations.
* `golangci-lint` for local linting.

## Configuration

The application reads environment variables from:

```text
configs/.env
```

Create a local config from the example file:

```bash
cp configs/example.env configs/.env
```

The local `configs/.env` file should not be committed.

Minimal configuration:

```env
APP_VERSION=0.1.0-dev
LOG_LEVEL=debug

DATABASE_URL=postgres://finance_tracker:finance_tracker@localhost:5432/finance_tracker?sslmode=disable
API_AUTH_TOKEN=change-me-to-a-long-random-token

DATA_PATH=~/.config/waybar/payments.json
DEPOSITS_DATA_PATH=~/.config/waybar/deposits.json

TELEGRAM_BOT_TOKEN=
TELEGRAM_USER_ID=0
```

`API_AUTH_TOKEN` is required. The server fails fast on startup when this value is empty.

Generate a strong token on Linux/macOS:

```bash
openssl rand -hex 32
```

Generate a strong token on Windows PowerShell:

```powershell
[Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32)).ToLower()
```

Put the generated value into `configs/.env`:

```env
API_AUTH_TOKEN=<generated-token>
```

## Database Setup

Create a local PostgreSQL database and user:

```sql
CREATE USER finance_tracker WITH PASSWORD 'finance_tracker';
CREATE DATABASE finance_tracker OWNER finance_tracker;
```

Run migrations:

```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 \
  -dir migrations \
  postgres "postgres://finance_tracker:finance_tracker@localhost:5432/finance_tracker?sslmode=disable" \
  up
```

Check migration status:

```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 \
  -dir migrations \
  postgres "postgres://finance_tracker:finance_tracker@localhost:5432/finance_tracker?sslmode=disable" \
  status
```

## Running the API

Before starting the API, make sure `configs/.env` exists and contains `API_AUTH_TOKEN`.

Start the server:

```bash
go run ./cmd/server --addr :8080
```

Or pass a database URL explicitly:

```bash
go run ./cmd/server \
  --addr :8080 \
  --database-url "postgres://finance_tracker:finance_tracker@localhost:5432/finance_tracker?sslmode=disable"
```

Public endpoints:

```text
GET /health
GET /ready
```

Protected API routes require the `Authorization` header:

```bash
curl -H "Authorization: Bearer <API_AUTH_TOKEN>" \
  http://localhost:8080/api/accounts
```

## API Overview

Accounts:

```text
GET    /api/accounts
POST   /api/accounts
GET    /api/accounts/{id}
PATCH  /api/accounts/{id}
POST   /api/accounts/{id}/archive
GET    /api/accounts/{id}/balance
```

Transactions:

```text
GET    /api/transactions
POST   /api/transactions
GET    /api/transactions/{id}
DELETE /api/transactions/{id}
```

Transfers:

```text
POST /api/transfers
```

Interest rules and accruals:

```text
GET   /api/accounts/{id}/interest-rules
POST  /api/accounts/{id}/interest-rules
PATCH /api/interest-rules/{id}
POST  /api/accounts/{id}/accrue-interest
```

Example: create an account:

```bash
curl -X POST http://localhost:8080/api/accounts \
  -H "Authorization: Bearer <API_AUTH_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Main Savings",
    "bank": "Yandex",
    "type": "savings",
    "currency": "RUB",
    "opened_at": "2026-05-01"
  }'
```

Example: get account balance:

```bash
curl -H "Authorization: Bearer <API_AUTH_TOKEN>" \
  http://localhost:8080/api/accounts/<account-id>/balance
```

Example: accrue interest for an account:

```bash
curl -X POST http://localhost:8080/api/accounts/<account-id>/accrue-interest \
  -H "Authorization: Bearer <API_AUTH_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "date": "2026-05-06"
  }'
```

If the request body is empty, the endpoint uses default accrual behavior.

## Interest Rules

Interest rules define how interest should be accrued for an account.

Important behavior:

* The rule must belong to the target account.
* When no `rule_id` is provided for manual accrual, the API selects the latest active rule that applies to the requested accrual date.
* Balance used for accrual is calculated only from transactions with `occurred_at` on or before the accrual date.
* Duplicate accruals for the same account, rule, and date are skipped.
* Promo rate and promo end date must be set together.
* Existing promo settings can be cleared with `null` or an empty promo end date.

Example: clear a promo rate:

```bash
curl -X PATCH http://localhost:8080/api/interest-rules/<rule-id> \
  -H "Authorization: Bearer <API_AUTH_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "promo_rate_bps": null
  }'
```

## Validation Notes

The HTTP API validates user input before writing to PostgreSQL:

* Currency must be exactly three uppercase Latin letters, for example `RUB`, `USD`, or `EUR`.
* Unknown JSON fields are rejected.
* Trailing JSON data is rejected.
* Invalid enum values are rejected.
* Invalid interest rule date ranges are rejected.
* Missing resources return `404` instead of silently returning empty data.

## Development

Download dependencies:

```bash
go mod download
```

Run all tests and lint:

```bash
make check-race
```

or without race:

```bash
make check
```

Check formatting:

```bash
gofmt -l $(git ls-files '*.go')
```

Build binaries:

```bash
go build ./cmd/...
```

## CI

The GitHub Actions CI pipeline runs:

* `go test ./...` on Linux and Windows.
* `go test -race ./...` on Linux.
* `golangci-lint`.
* `go build ./cmd/...`.
* PostgreSQL migration checks.
* `go mod tidy` verification.
* `gofmt` check on Linux.

## Security

All `/api/*` routes require Bearer token authentication:

```http
Authorization: Bearer <API_AUTH_TOKEN>
```

Do not commit real secrets. Keep local secrets in `configs/.env` and commit only `configs/example.env`.

Public routes:

```text
GET /health
GET /ready
```

Protected routes:

```text
/api/*
```

## License

No license is currently specified.
