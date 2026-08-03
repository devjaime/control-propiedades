-- +goose Up
-- +goose StatementBegin
ALTER TABLE leases
    ADD COLUMN renewal_review_days smallint NOT NULL DEFAULT 45
        CHECK (renewal_review_days BETWEEN 0 AND 365),
    ADD COLUMN adjustment_effective_on date,
    ADD CONSTRAINT leases_adjustment_effective_check CHECK (
        adjustment_effective_on IS NULL OR adjustment_method = 'ipc'
    );

ALTER TABLE lease_payment_schedules
    ADD COLUMN schedule_kind text NOT NULL DEFAULT 'contractual'
        CHECK (schedule_kind IN ('contractual', 'special_agreement')),
    ADD COLUMN agreement_notes text;

UPDATE lease_payment_schedules schedule
SET schedule_kind = CASE
    WHEN (SELECT count(*) FROM lease_payment_schedules other WHERE other.lease_id=schedule.lease_id) > 1
        THEN 'special_agreement'
    ELSE 'contractual'
END;

CREATE TABLE lease_amendments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    lease_id uuid NOT NULL REFERENCES leases(id) ON DELETE RESTRICT,
    effective_on date NOT NULL,
    reason text NOT NULL CHECK (length(trim(reason)) BETWEEN 3 AND 1000),
    previous_state jsonb NOT NULL,
    new_state jsonb NOT NULL,
    approved_by uuid NOT NULL REFERENCES users(id),
    approved_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX lease_amendments_history_idx
    ON lease_amendments (organization_id, lease_id, effective_on DESC, created_at DESC);

ALTER TABLE notifications DROP CONSTRAINT notifications_notification_type_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_notification_type_check CHECK (
    notification_type IN (
        'payment_review', 'charge_due', 'charge_overdue', 'rent_completed', 'tenant_report',
        'lease_expiry', 'vacate_notice', 'rent_adjustment', 'maintenance_due', 'exit_inspection'
    )
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notifications DROP CONSTRAINT notifications_notification_type_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_notification_type_check CHECK (
    notification_type IN (
        'payment_review', 'charge_due', 'charge_overdue', 'rent_completed', 'tenant_report',
        'lease_expiry', 'vacate_notice', 'rent_adjustment', 'maintenance_due'
    )
);
DROP TABLE lease_amendments;
ALTER TABLE lease_payment_schedules
    DROP COLUMN agreement_notes,
    DROP COLUMN schedule_kind;
ALTER TABLE leases
    DROP CONSTRAINT leases_adjustment_effective_check,
    DROP COLUMN adjustment_effective_on,
    DROP COLUMN renewal_review_days;
-- +goose StatementEnd
