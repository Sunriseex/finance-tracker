# Operations Runbook

This page is the operational entrypoint.

## Health Checks

* `GET /health`: process is alive.
* `GET /ready`: dependencies are ready.
* `GET /metrics`: expvar metrics, including auth counters.

## Normal Deploy Checklist

1. Confirm CI is green.
2. Apply database migrations.
3. Start or roll the backend.
4. Check `/ready`.
5. Check `/metrics`.
6. Review logs for startup errors.

## Auth Checks

After deploying auth changes:

1. Run setup/login in a test environment.
2. Verify refresh rotation.
3. Verify logout revokes the refresh session.
4. Verify password change revokes all sessions.
5. Verify `auth_audit_events` receives records.
6. Verify `capitalflow_auth_events_total` changes in `/metrics`.

## Incident Pages

* [Auth Incident Response](Auth-Incident-Response)
* [Auth Observability](Auth-Observability)
