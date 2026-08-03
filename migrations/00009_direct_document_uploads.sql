-- +goose Up
-- +goose StatementBegin
CREATE TABLE document_uploads (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    incident_id uuid REFERENCES incidents(id) ON DELETE SET NULL,
    uploaded_by uuid NOT NULL REFERENCES users(id),
    storage_key text NOT NULL UNIQUE,
    display_name text NOT NULL CHECK (length(trim(display_name)) BETWEEN 1 AND 240),
    original_name text NOT NULL CHECK (length(trim(original_name)) BETWEEN 1 AND 255),
    document_type text NOT NULL CHECK (document_type IN (
        'deed', 'domain_registration', 'valuation', 'lease', 'annex',
        'inventory', 'bank_receipt', 'rent_receipt', 'mortgage',
        'property_tax', 'insurance', 'common_expense', 'utility', 'quote',
        'invoice', 'warranty', 'photo', 'inspection', 'certificate', 'other'
    )),
    mime_type text NOT NULL CHECK (mime_type IN ('application/pdf', 'image/jpeg', 'image/png', 'image/webp')),
    size_bytes bigint NOT NULL CHECK (size_bytes > 0),
    sha256 text NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    tags text[] NOT NULL DEFAULT '{}',
    expires_at timestamptz NOT NULL,
    completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX document_uploads_pending_idx
    ON document_uploads (organization_id, expires_at)
    WHERE completed_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE document_uploads;
-- +goose StatementEnd
