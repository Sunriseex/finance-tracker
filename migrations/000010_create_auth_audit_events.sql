-- +goose Up
CREATE TABLE auth_audit_events (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    success BOOLEAN NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX auth_audit_events_created_at_idx ON auth_audit_events (created_at DESC);
CREATE INDEX auth_audit_events_user_id_idx ON auth_audit_events (user_id);
CREATE INDEX auth_audit_events_event_type_idx ON auth_audit_events (event_type);

-- +goose Down
DROP TABLE IF EXISTS auth_audit_events;
