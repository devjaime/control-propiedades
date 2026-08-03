-- +goose Up
-- +goose StatementBegin
CREATE TABLE tenants (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 2 AND 200),
    identifier text CHECK (identifier IS NULL OR length(trim(identifier)) BETWEEN 3 AND 40),
    email text CHECK (email IS NULL OR length(trim(email)) BETWEEN 3 AND 320),
    phone text CHECK (phone IS NULL OR length(trim(phone)) BETWEEN 5 AND 40),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'archived')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tenants_organization_name_idx ON tenants (organization_id, lower(name));

CREATE TABLE leases (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE RESTRICT,
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    starts_on date NOT NULL,
    ends_on date,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'ending', 'ended', 'terminated', 'archived')),
    rent_amount_minor bigint NOT NULL CHECK (rent_amount_minor > 0),
    currency_code varchar(3) NOT NULL DEFAULT 'CLP' CHECK (currency_code ~ '^[A-Z]{3}$'),
    payment_day smallint NOT NULL CHECK (payment_day BETWEEN 1 AND 28),
    service_responsibility jsonb NOT NULL DEFAULT '{}'::jsonb,
    special_terms text,
    created_by uuid NOT NULL REFERENCES users(id),
    updated_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    CHECK (ends_on IS NULL OR ends_on >= starts_on)
);
CREATE UNIQUE INDEX leases_one_active_per_property_idx
    ON leases (organization_id, property_id) WHERE status IN ('active', 'ending');
CREATE INDEX leases_tenant_idx ON leases (organization_id, tenant_id, status);

CREATE TABLE charges (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE RESTRICT,
    lease_id uuid NOT NULL REFERENCES leases(id) ON DELETE RESTRICT,
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    period varchar(7) NOT NULL CHECK (period ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    charge_type text NOT NULL CHECK (charge_type IN ('rent', 'electricity', 'water', 'trash', 'common_expense', 'adjustment', 'other')),
    concept text NOT NULL CHECK (length(trim(concept)) BETWEEN 2 AND 200),
    due_date date NOT NULL,
    original_amount_minor bigint NOT NULL CHECK (original_amount_minor > 0),
    balance_minor bigint NOT NULL CHECK (balance_minor >= 0 AND balance_minor <= original_amount_minor),
    currency_code varchar(3) NOT NULL DEFAULT 'CLP' CHECK (currency_code ~ '^[A-Z]{3}$'),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'partially_paid', 'paid', 'overdue', 'disputed', 'cancelled')),
    origin text NOT NULL DEFAULT 'manual' CHECK (origin IN ('automatic', 'manual')),
    notes text,
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0)
);
CREATE UNIQUE INDEX charges_lease_period_type_idx ON charges (lease_id, period, charge_type);
CREATE INDEX charges_review_idx ON charges (organization_id, status, due_date);
CREATE INDEX charges_property_period_idx ON charges (organization_id, property_id, period);

CREATE TABLE payments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE RESTRICT,
    lease_id uuid NOT NULL REFERENCES leases(id) ON DELETE RESTRICT,
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    payer_name text NOT NULL CHECK (length(trim(payer_name)) BETWEEN 2 AND 200),
    payment_date date NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    currency_code varchar(3) NOT NULL DEFAULT 'CLP' CHECK (currency_code ~ '^[A-Z]{3}$'),
    payment_method text NOT NULL CHECK (payment_method IN ('bank_transfer', 'deposit', 'cash', 'other')),
    bank_reference text,
    receiver_account text,
    supporting_document_id uuid REFERENCES documents(id) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'pending_reconciliation' CHECK (status IN ('received', 'pending_reconciliation', 'reconciled', 'rejected', 'reversed', 'duplicate')),
    timing_status text NOT NULL DEFAULT 'unclassified' CHECK (timing_status IN ('on_time', 'late', 'unclassified')),
    observations text,
    registered_by uuid NOT NULL REFERENCES users(id),
    confirmed_by uuid REFERENCES users(id),
    confirmed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0)
);
CREATE INDEX payments_property_date_idx ON payments (organization_id, property_id, payment_date DESC);
CREATE INDEX payments_review_idx ON payments (organization_id, status, created_at);

CREATE TABLE payment_allocations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    payment_id uuid NOT NULL REFERENCES payments(id) ON DELETE RESTRICT,
    charge_id uuid NOT NULL REFERENCES charges(id) ON DELETE RESTRICT,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    status text NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed', 'confirmed', 'reversed')),
    confirmed_by uuid REFERENCES users(id),
    confirmed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT payment_allocations_payment_charge_key UNIQUE (payment_id, charge_id)
);
CREATE INDEX payment_allocations_charge_idx ON payment_allocations (charge_id, status);

CREATE TABLE rental_observations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    lease_id uuid REFERENCES leases(id) ON DELETE SET NULL,
    charge_id uuid REFERENCES charges(id) ON DELETE SET NULL,
    payment_id uuid REFERENCES payments(id) ON DELETE SET NULL,
    category text NOT NULL CHECK (category IN ('payment', 'tenant_report', 'property', 'service', 'general')),
    content text NOT NULL CHECK (length(trim(content)) BETWEEN 2 AND 2000),
    observed_at timestamptz NOT NULL DEFAULT now(),
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX rental_observations_property_idx ON rental_observations (organization_id, property_id, observed_at DESC);

CREATE TABLE notifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid REFERENCES properties(id) ON DELETE CASCADE,
    notification_type text NOT NULL CHECK (notification_type IN ('payment_review', 'charge_due', 'charge_overdue', 'rent_completed', 'tenant_report')),
    title text NOT NULL,
    message text NOT NULL,
    entity_type text,
    entity_id uuid,
    due_at timestamptz,
    status text NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'read', 'dismissed')),
    dedup_key text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    read_at timestamptz,
    CONSTRAINT notifications_organization_dedup_key UNIQUE (organization_id, dedup_key)
);
CREATE INDEX notifications_inbox_idx ON notifications (organization_id, status, due_at, created_at DESC);

CREATE TABLE receipt_sequences (
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    sequence_year integer NOT NULL CHECK (sequence_year BETWEEN 2020 AND 2200),
    last_value bigint NOT NULL DEFAULT 0 CHECK (last_value >= 0),
    PRIMARY KEY (organization_id, sequence_year)
);

CREATE TABLE rent_receipts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE RESTRICT,
    lease_id uuid NOT NULL REFERENCES leases(id) ON DELETE RESTRICT,
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    charge_id uuid NOT NULL REFERENCES charges(id) ON DELETE RESTRICT,
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
    folio text NOT NULL,
    public_token text NOT NULL,
    verification_code text NOT NULL,
    snapshot jsonb NOT NULL,
    pdf_sha256 char(64) NOT NULL CHECK (pdf_sha256 ~ '^[a-f0-9]{64}$'),
    template_version text NOT NULL,
    signature_name text NOT NULL,
    signed_at timestamptz NOT NULL,
    status text NOT NULL DEFAULT 'valid' CHECK (status IN ('valid', 'replaced', 'void')),
    issued_by uuid NOT NULL REFERENCES users(id),
    issued_at timestamptz NOT NULL DEFAULT now(),
    replaced_by uuid REFERENCES rent_receipts(id),
    CONSTRAINT rent_receipts_folio_key UNIQUE (organization_id, folio),
    CONSTRAINT rent_receipts_public_token_key UNIQUE (public_token),
    CONSTRAINT rent_receipts_verification_code_key UNIQUE (verification_code)
);
CREATE UNIQUE INDEX rent_receipts_active_charge_idx ON rent_receipts (charge_id) WHERE status = 'valid';
CREATE INDEX rent_receipts_property_idx ON rent_receipts (organization_id, property_id, issued_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE rent_receipts;
DROP TABLE receipt_sequences;
DROP TABLE notifications;
DROP TABLE rental_observations;
DROP TABLE payment_allocations;
DROP TABLE payments;
DROP TABLE charges;
DROP TABLE leases;
DROP TABLE tenants;
-- +goose StatementEnd
