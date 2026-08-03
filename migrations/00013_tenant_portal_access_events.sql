-- +goose Up
-- +goose StatementBegin
CREATE TABLE tenant_portal_access_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    portal_link_id uuid NOT NULL REFERENCES tenant_portal_links(id) ON DELETE RESTRICT,
    credential_kind text NOT NULL CHECK (credential_kind IN ('owner', 'tenant', 'legacy_key')),
    accessed_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX tenant_portal_access_events_link_time_idx
    ON tenant_portal_access_events (portal_link_id, accessed_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE tenant_portal_access_events;
-- +goose StatementEnd
