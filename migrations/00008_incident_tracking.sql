-- +goose Up
-- +goose StatementBegin
CREATE TABLE incidents (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    reference text NOT NULL,
    title text NOT NULL CHECK (length(trim(title)) BETWEEN 3 AND 240),
    summary text NOT NULL CHECK (length(trim(summary)) BETWEEN 3 AND 10000),
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
    priority text NOT NULL CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    event_date date NOT NULL,
    categories text[] NOT NULL DEFAULT '{}',
    tags text[] NOT NULL DEFAULT '{}',
    habitability text NOT NULL DEFAULT 'unaffected' CHECK (habitability IN ('unaffected', 'partially_affected', 'uninhabitable')),
    created_by uuid NOT NULL REFERENCES users(id),
    updated_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    UNIQUE (organization_id, reference)
);
CREATE INDEX incidents_property_status_idx ON incidents (organization_id, property_id, status, event_date DESC);

CREATE TABLE incident_tasks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id uuid NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    title text NOT NULL CHECK (length(trim(title)) BETWEEN 3 AND 500),
    priority text NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
    due_on date,
    completed_at timestamptz,
    created_by uuid NOT NULL REFERENCES users(id),
    updated_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    CHECK ((status = 'completed' AND completed_at IS NOT NULL) OR (status <> 'completed' AND completed_at IS NULL))
);
CREATE INDEX incident_tasks_incident_status_idx ON incident_tasks (incident_id, status, priority);

CREATE TABLE incident_updates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id uuid NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    update_type text NOT NULL CHECK (update_type IN ('note', 'tenant_contact', 'builder_claim', 'insurance_claim', 'inspection', 'status_change')),
    content text NOT NULL CHECK (length(trim(content)) BETWEEN 2 AND 10000),
    occurred_at timestamptz NOT NULL DEFAULT now(),
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX incident_updates_timeline_idx ON incident_updates (incident_id, occurred_at DESC);

CREATE TABLE incident_documents (
    incident_id uuid NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (incident_id, document_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE incident_documents;
DROP TABLE incident_updates;
DROP TABLE incident_tasks;
DROP TABLE incidents;
-- +goose StatementEnd
