# Getting Started

This page is a compact setup map. Use `README.md` for copy-paste commands.

## Requirements

* Go 1.26.2 or newer.
* PostgreSQL 17 or compatible.
* Goose for migrations.
* Node.js and npm for the Web UI.

## Local Flow

1. Copy `configs/example.env` to `configs/.env`.
2. Set `DATABASE_URL`, `API_AUTH_TOKEN`, and JWT-related auth settings.
3. Run database migrations from `migrations/`.
4. Start the API with `go run ./cmd/server --addr :8080`.
5. Start the Web UI from `web/`.

## Useful Checks

```bash
go test ./...
cd web && npm run lint && npm run build
```

## Next Pages

* [Architecture](Architecture)
* [API](API)
* [Database and Migrations](Database-and-Migrations)
