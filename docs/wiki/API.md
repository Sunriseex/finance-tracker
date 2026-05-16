# API

The API contract is maintained in `docs/openapi.yaml`.

## Main Route Groups

* `/auth/*`: setup, login, refresh, logout.
* `/api/v1/auth/*`: password and session management.
* `/api/v1/settings/profile`: user profile.
* `/api/v1/accounts/*`: accounts and balances.
* `/api/v1/transactions/*`: transactions.
* `/api/v1/transfers`: transfers.
* `/api/v1/categories`: categories.
* `/api/v1/currency-rates`: currency rates.

## Authentication

Most `/api/v1/*` routes require an `Authorization: Bearer <access_token>` header.

Refresh and logout use the secure refresh cookie. Auth JSON responses do not include refresh tokens.

See [Auth Security Model](Auth-Security-Model).

## Contract Checks

OpenAPI linting runs in CI through Redocly against `docs/openapi.yaml`.
