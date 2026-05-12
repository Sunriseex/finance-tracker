# Auth Observability

Auth events are exported through `GET /metrics` in the `capitalflow_auth_events_total` expvar map.

Metric keys use:

```text
event_type=<event>,success=<true|false>,reason=<reason>
```

## Important Events

* `login_failed`
* `login_success`
* `refresh_failed`
* `refresh_success`
* `refresh_reuse_detected`
* `logout`
* `change_password_failed`
* `change_password_success`
* `sessions_listed`
* `session_revoked`
* `session_revoke_failed`

## Alert Rules

Alert on:

* `refresh_reuse_detected` count greater than 0 in 5 minutes.
* `login_failed` with `reason=account_locked` greater than 3 in 10 minutes.
* `change_password_failed` greater than 5 in 10 minutes.
* `session_revoke_failed` greater than 5 in 10 minutes.
* `/metrics`, `/health`, or `/ready` unavailable for 2 minutes.

## Audit Query

Use the audit table for incident details:

```sql
SELECT created_at, event_type, user_id, email, success, reason
FROM auth_audit_events
WHERE created_at >= now() - interval '1 hour'
ORDER BY created_at DESC;
```
