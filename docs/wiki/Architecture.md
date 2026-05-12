# Architecture

CapitalFlow is a layered Go backend with a React Web UI.

## Backend Layout

* `cmd/server`: HTTP API entrypoint.
* `internal/http`: routing, handlers, DTOs, and middleware.
* `internal/services`: business logic.
* `internal/repository`: repository contracts.
* `internal/postgres`: PostgreSQL implementations.
* `internal/models`: shared domain models.
* `migrations`: database schema changes.

## Request Flow

HTTP requests enter through Chi routes, pass through middleware, then call handlers. Handlers decode DTOs and delegate business rules to services. Services use repository interfaces and do not depend on HTTP-specific response types.

## Auth Flow

Auth uses short-lived JWT access tokens and refresh tokens. Refresh token records are stored in PostgreSQL and are checked by JWT middleware through the session ID embedded in the access token.

See [Auth Security Model](Auth-Security-Model).

## Frontend

The Web UI lives in `web/` and uses React, Vite, TypeScript, TanStack Query, and Recharts.

See [Web UI](Web-UI).
