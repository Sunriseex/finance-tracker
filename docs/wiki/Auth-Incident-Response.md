# Auth Incident Response

Use this runbook for auth incidents.

## Refresh Token Reuse

Trigger:

* `refresh_reuse_detected` appears in `/metrics` or `auth_audit_events`.

Immediate actions:

1. Identify affected `user_id` from `auth_audit_events`.
2. Confirm all active refresh tokens for that user are revoked.
3. Ask the user to sign in again.
4. Review recent `login_failed`, `refresh_failed`, and `session_revoked` events.
5. Rotate credentials if account compromise is suspected.

Query:

```sql
SELECT *
FROM auth_audit_events
WHERE event_type = 'refresh_reuse_detected'
ORDER BY created_at DESC;
```

## Lockout Spike

Trigger:

* `login_failed,reason=account_locked` exceeds threshold.

Immediate actions:

1. Group events by email and user ID.
2. Check whether the spike affects one account or many accounts.
3. If many accounts are affected, inspect IP-level rate limit logs.
4. If one account is affected, contact the user and verify recent activity.
5. Keep the lockout in place unless there is a confirmed false positive.

## Password Change Abuse

Trigger:

* `change_password_failed` exceeds threshold.
* Unexpected `change_password_success` is reported.

Immediate actions:

1. Check `auth_audit_events` for the user ID.
2. Verify all refresh sessions were revoked after password change.
3. Ask the user to sign in again and rotate password if needed.
4. Review session list and revoke suspicious sessions.

## Session Revocation Issues

Trigger:

* `session_revoke_failed` exceeds threshold.

Immediate actions:

1. Confirm session IDs belong to the authenticated user.
2. Check whether clients are retrying stale session IDs.
3. Review API errors and idempotency records.
