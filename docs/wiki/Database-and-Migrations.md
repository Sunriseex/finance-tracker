# Database and Migrations

PostgreSQL is the source of truth for runtime data.

## Migration Tool

The project uses Goose migrations in `migrations/`.

## Core Auth Tables

* `users`: user identity, password hash, primary currency, email verification fields, lockout state.
* `refresh_tokens`: refresh session records and revocation state.
* `auth_audit_events`: auth security event trail.
* `idempotency_keys`: mutation idempotency records.

## Operational Rules

* Apply migrations before starting a new backend build.
* Do not edit existing applied migrations.
* Add new schema changes as new numbered migration files.
* Keep repository models and scan queries in sync with schema changes.

## Useful Query

```sql
SELECT version_id, is_applied, tstamp
FROM goose_db_version
ORDER BY tstamp DESC;
```
