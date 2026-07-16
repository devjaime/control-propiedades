-- +goose Up
-- +goose StatementBegin
CREATE TABLE organizations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    slug text NOT NULL CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'closed')),
    timezone text NOT NULL DEFAULT 'America/Santiago',
    default_currency varchar(3) NOT NULL DEFAULT 'CLP' CHECK (default_currency ~ '^[A-Z]{3}$'),
    country_code char(2) NOT NULL DEFAULT 'CL' CHECK (country_code ~ '^[A-Z]{2}$'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    CONSTRAINT organizations_slug_key UNIQUE (slug)
);

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    email text NOT NULL CHECK (length(trim(email)) BETWEEN 3 AND 320),
    password_hash text,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'disabled')),
    email_verified_at timestamptz,
    last_login_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0)
);
CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email));

CREATE TABLE memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    user_id uuid NOT NULL REFERENCES users(id),
    role text NOT NULL CHECK (role IN ('organization_admin', 'property_manager', 'owner', 'viewer')),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('invited', 'active', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    CONSTRAINT memberships_organization_user_key UNIQUE (organization_id, user_id)
);
CREATE INDEX memberships_user_id_idx ON memberships (user_id);

CREATE TABLE properties (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    slug text NOT NULL CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    property_type text NOT NULL CHECK (property_type IN ('house', 'apartment', 'land', 'commercial', 'other')),
    usage text NOT NULL CHECK (usage IN ('primary_residence', 'rental', 'second_home', 'vacant', 'renovation', 'for_sale', 'acquisition')),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'inactive', 'archived')),
    address_line text NOT NULL,
    commune text NOT NULL,
    region text NOT NULL,
    country_code char(2) NOT NULL DEFAULT 'CL' CHECK (country_code ~ '^[A-Z]{2}$'),
    notes text,
    created_by uuid REFERENCES users(id),
    updated_by uuid REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    CONSTRAINT properties_organization_slug_key UNIQUE (organization_id, slug)
);
CREATE INDEX properties_organization_status_idx ON properties (organization_id, status);

CREATE TABLE audit_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid REFERENCES organizations(id),
    actor_type text NOT NULL CHECK (actor_type IN ('user', 'agent', 'system')),
    actor_id uuid,
    action text NOT NULL,
    entity_type text NOT NULL,
    entity_id uuid,
    previous_state jsonb,
    new_state jsonb,
    reason text,
    correlation_id text,
    ip_address inet,
    technical_context jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_events_organization_occurred_idx
    ON audit_events (organization_id, occurred_at DESC);
CREATE INDEX audit_events_entity_idx
    ON audit_events (organization_id, entity_type, entity_id, occurred_at DESC);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE audit_events;
DROP TABLE properties;
DROP TABLE memberships;
DROP TABLE users;
DROP TABLE organizations;
-- +goose StatementEnd
