-- +goose Up
-- +goose StatementBegin
ALTER TABLE incidents
    ADD COLUMN tenant_public_summary text
        CHECK (tenant_public_summary IS NULL OR length(trim(tenant_public_summary)) BETWEEN 3 AND 4000);

ALTER TABLE incident_tasks
    ADD COLUMN visible_to_tenant boolean NOT NULL DEFAULT false;

ALTER TABLE incident_updates
    ADD COLUMN visible_to_tenant boolean NOT NULL DEFAULT false;

CREATE TABLE tenant_portal_links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    lease_id uuid REFERENCES leases(id) ON DELETE SET NULL,
    public_token text NOT NULL UNIQUE CHECK (length(public_token) >= 32),
    access_key_hash text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'expired')),
    failed_attempts smallint NOT NULL DEFAULT 0 CHECK (failed_attempts BETWEEN 0 AND 100),
    locked_until timestamptz,
    expires_at timestamptz,
    last_accessed_at timestamptz,
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    CHECK (expires_at IS NULL OR expires_at > created_at)
);
CREATE UNIQUE INDEX tenant_portal_one_active_property_idx
    ON tenant_portal_links (organization_id, property_id) WHERE status = 'active';
CREATE INDEX tenant_portal_lookup_idx
    ON tenant_portal_links (public_token, status, expires_at);

CREATE TABLE api_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 3 AND 120),
    token_prefix varchar(16) NOT NULL,
    token_hash bytea NOT NULL UNIQUE,
    scopes text[] NOT NULL DEFAULT '{}',
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    expires_at timestamptz,
    last_used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    created_by uuid NOT NULL REFERENCES users(id),
    CHECK (expires_at IS NULL OR expires_at > created_at),
    CHECK (scopes <@ ARRAY['property:read', 'incident:read', 'incident:update', 'rent:read']::text[])
);
CREATE INDEX api_tokens_active_lookup_idx
    ON api_tokens (token_hash, status, expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE api_tokens;
DROP TABLE tenant_portal_links;
ALTER TABLE incident_updates DROP COLUMN visible_to_tenant;
ALTER TABLE incident_tasks DROP COLUMN visible_to_tenant;
ALTER TABLE incidents DROP COLUMN tenant_public_summary;
-- +goose StatementEnd
