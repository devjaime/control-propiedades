-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    CHECK (expires_at > created_at)
);
CREATE INDEX sessions_active_token_idx
    ON sessions (token_hash, expires_at)
    WHERE revoked_at IS NULL;
CREATE INDEX sessions_user_idx ON sessions (user_id, created_at DESC);

CREATE TABLE documents (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    document_type text NOT NULL CHECK (document_type IN (
        'deed', 'domain_registration', 'valuation', 'lease', 'annex',
        'inventory', 'bank_receipt', 'rent_receipt', 'mortgage',
        'property_tax', 'insurance', 'common_expense', 'utility',
        'quote', 'invoice', 'warranty', 'photo', 'inspection',
        'certificate', 'other'
    )),
    display_name text NOT NULL CHECK (length(trim(display_name)) BETWEEN 1 AND 240),
    original_name text NOT NULL CHECK (length(trim(original_name)) BETWEEN 1 AND 255),
    storage_key text NOT NULL UNIQUE,
    mime_type text NOT NULL,
    size_bytes bigint NOT NULL CHECK (size_bytes > 0),
    sha256 char(64) NOT NULL CHECK (sha256 ~ '^[a-f0-9]{64}$'),
    tags text[] NOT NULL DEFAULT '{}',
    status text NOT NULL DEFAULT 'uploaded' CHECK (status IN (
        'uploaded', 'pending_review', 'verified', 'rejected', 'duplicate', 'archived'
    )),
    uploaded_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0)
);
CREATE INDEX documents_organization_created_idx
    ON documents (organization_id, created_at DESC);
CREATE INDEX documents_property_created_idx
    ON documents (organization_id, property_id, created_at DESC);
CREATE INDEX documents_type_idx
    ON documents (organization_id, document_type, created_at DESC);
CREATE INDEX documents_hash_idx
    ON documents (organization_id, sha256);
CREATE INDEX documents_tags_gin_idx ON documents USING gin (tags);
CREATE INDEX documents_names_trgm_idx ON documents USING gin (
    (lower(display_name || ' ' || original_name)) gin_trgm_ops
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE documents;
DROP TABLE sessions;
DROP EXTENSION IF EXISTS pg_trgm;
-- +goose StatementEnd
