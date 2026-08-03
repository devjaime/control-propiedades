-- +goose Up
-- +goose StatementBegin
CREATE TABLE tenant_applications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    tenant_id uuid REFERENCES tenants(id) ON DELETE SET NULL,
    full_name text NOT NULL CHECK (length(trim(full_name)) BETWEEN 2 AND 200),
    identifier text NOT NULL CHECK (length(trim(identifier)) BETWEEN 3 AND 40),
    nationality text NOT NULL DEFAULT 'Chilena',
    marital_status text NOT NULL DEFAULT '',
    profession text NOT NULL DEFAULT '',
    current_address text NOT NULL DEFAULT '',
    email text NOT NULL DEFAULT '',
    phone text NOT NULL DEFAULT '',
    employer text NOT NULL DEFAULT '',
    monthly_income_minor bigint CHECK (monthly_income_minor IS NULL OR monthly_income_minor >= 0),
    authorized_occupants text NOT NULL DEFAULT '',
    notes text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'under_review', 'approved', 'rejected', 'contracted', 'archived')),
    commercial_report_provenance_confirmed boolean NOT NULL DEFAULT false,
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0)
);
CREATE INDEX tenant_applications_property_idx
    ON tenant_applications (organization_id, property_id, created_at DESC);

CREATE TABLE tenant_application_documents (
    application_id uuid NOT NULL REFERENCES tenant_applications(id) ON DELETE CASCADE,
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
    document_kind text NOT NULL CHECK (document_kind IN (
        'commercial_report', 'identity', 'income_proof', 'employment',
        'guarantor', 'insurance', 'other'
    )),
    linked_by uuid NOT NULL REFERENCES users(id),
    linked_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (application_id, document_id)
);
CREATE INDEX tenant_application_documents_kind_idx
    ON tenant_application_documents (application_id, document_kind, linked_at DESC);

CREATE TABLE lease_contract_drafts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    application_id uuid NOT NULL REFERENCES tenant_applications(id) ON DELETE RESTRICT,
    lease_id uuid REFERENCES leases(id) ON DELETE SET NULL,
    draft_version integer NOT NULL CHECK (draft_version > 0),
    template_version text NOT NULL,
    legal_basis_version text NOT NULL,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN (
        'draft', 'reviewed', 'approved_for_signature', 'signed_notarized', 'superseded', 'void'
    )),
    snapshot jsonb NOT NULL,
    pdf_document_id uuid NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
    docx_document_id uuid NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
    signed_document_id uuid REFERENCES documents(id) ON DELETE RESTRICT,
    pdf_sha256 char(64) NOT NULL CHECK (pdf_sha256 ~ '^[a-f0-9]{64}$'),
    docx_sha256 char(64) NOT NULL CHECK (docx_sha256 ~ '^[a-f0-9]{64}$'),
    legal_reviewed_by text,
    legal_reviewed_at timestamptz,
    notarized_verified_by uuid REFERENCES users(id),
    notarized_verified_at timestamptz,
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (application_id, draft_version),
    CHECK ((status = 'signed_notarized') = (signed_document_id IS NOT NULL))
);
CREATE INDEX lease_contract_drafts_property_idx
    ON lease_contract_drafts (organization_id, property_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE lease_contract_drafts;
DROP TABLE tenant_application_documents;
DROP TABLE tenant_applications;
-- +goose StatementEnd
