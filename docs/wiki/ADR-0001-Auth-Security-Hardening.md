# ADR 0001: Auth Security Hardening

## Status

Accepted.

## Context

CapitalFlow needed stronger auth behavior for refresh token safety, password policy, session control, auditability, and operations.

## Decision

Use short-lived JWT access tokens with PostgreSQL-backed refresh sessions.

Implement:

* refresh token rotation
* refresh token reuse detection
* full session family revocation on reuse
* secure refresh cookie fallback
* `zxcvbn` password policy
* progressive login lockout
* password change with logout from all sessions
* user-visible session listing and revocation
* auth audit events
* expvar auth counters and alert rules

## CSRF Decision

API mutations use Bearer access tokens. Refresh/logout support explicit JSON refresh tokens and a secure cookie fallback. The current model does not treat cookies as the only auth signal.

If refresh becomes cookie-only later, add explicit CSRF token checks.

## Consequences

* Access token validity depends on the referenced refresh session still being active.
* Revoking a refresh session also invalidates access tokens that reference that session.
* The audit table is the primary investigation source.
* `GET /metrics` is the lightweight metrics endpoint.
