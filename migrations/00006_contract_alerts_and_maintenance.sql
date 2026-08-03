-- +goose Up
-- +goose StatementBegin
ALTER TABLE leases
    ADD COLUMN renewal_notice_days smallint NOT NULL DEFAULT 60
        CHECK (renewal_notice_days BETWEEN 0 AND 365),
    ADD COLUMN vacate_notice_months smallint NOT NULL DEFAULT 1
        CHECK (vacate_notice_months BETWEEN 0 AND 24),
    ADD COLUMN vacate_notice_days smallint NOT NULL DEFAULT 0
        CHECK (vacate_notice_days BETWEEN 0 AND 90),
    ADD COLUMN adjustment_method text NOT NULL DEFAULT 'none'
        CHECK (adjustment_method IN ('none', 'ipc')),
    ADD COLUMN adjustment_frequency_months smallint
        CHECK (adjustment_frequency_months IS NULL OR adjustment_frequency_months BETWEEN 1 AND 36),
    ADD COLUMN next_adjustment_on date,
    ADD COLUMN ended_on date,
    ADD COLUMN end_reason text,
    ADD CONSTRAINT leases_adjustment_configuration_check CHECK (
        (adjustment_method = 'none' AND adjustment_frequency_months IS NULL AND next_adjustment_on IS NULL)
        OR
        (adjustment_method = 'ipc' AND adjustment_frequency_months IS NOT NULL AND next_adjustment_on IS NOT NULL)
    ),
    ADD CONSTRAINT leases_ended_on_check CHECK (ended_on IS NULL OR ended_on >= starts_on);

ALTER TABLE notifications DROP CONSTRAINT notifications_notification_type_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_notification_type_check CHECK (
    notification_type IN (
        'payment_review', 'charge_due', 'charge_overdue', 'rent_completed', 'tenant_report',
        'lease_expiry', 'vacate_notice', 'rent_adjustment', 'maintenance_due'
    )
);

CREATE TABLE maintenance_schedules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    property_id uuid NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    lease_id uuid REFERENCES leases(id) ON DELETE SET NULL,
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 2 AND 200),
    category text NOT NULL CHECK (category IN (
        'inspection', 'plumbing', 'electrical', 'gas', 'roof', 'painting',
        'appliance', 'garden', 'pest_control', 'insurance', 'other'
    )),
    frequency_months smallint NOT NULL CHECK (frequency_months BETWEEN 1 AND 120),
    next_due_on date NOT NULL,
    reminder_days smallint NOT NULL DEFAULT 15 CHECK (reminder_days BETWEEN 0 AND 365),
    last_completed_on date,
    notes text,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'archived')),
    created_by uuid NOT NULL REFERENCES users(id),
    updated_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0)
);
CREATE INDEX maintenance_schedules_due_idx
    ON maintenance_schedules (organization_id, property_id, status, next_due_on);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE maintenance_schedules;
ALTER TABLE notifications DROP CONSTRAINT notifications_notification_type_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_notification_type_check CHECK (
    notification_type IN ('payment_review', 'charge_due', 'charge_overdue', 'rent_completed', 'tenant_report')
);
ALTER TABLE leases
    DROP CONSTRAINT leases_ended_on_check,
    DROP CONSTRAINT leases_adjustment_configuration_check,
    DROP COLUMN end_reason,
    DROP COLUMN ended_on,
    DROP COLUMN next_adjustment_on,
    DROP COLUMN adjustment_frequency_months,
    DROP COLUMN adjustment_method,
    DROP COLUMN vacate_notice_days,
    DROP COLUMN vacate_notice_months,
    DROP COLUMN renewal_notice_days;
-- +goose StatementEnd
