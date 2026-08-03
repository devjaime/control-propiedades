-- +goose Up
-- +goose StatementBegin
ALTER TABLE leases
    ADD COLUMN payment_installments smallint NOT NULL DEFAULT 1
        CHECK (payment_installments BETWEEN 1 AND 12);

CREATE TABLE lease_payment_schedules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    lease_id uuid NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
    installment_number smallint NOT NULL CHECK (installment_number BETWEEN 1 AND 12),
    due_day smallint NOT NULL CHECK (due_day BETWEEN 1 AND 28),
    expected_amount_minor bigint NOT NULL CHECK (expected_amount_minor > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT lease_payment_schedules_number_key UNIQUE (lease_id, installment_number),
    CONSTRAINT lease_payment_schedules_day_key UNIQUE (lease_id, due_day)
);
CREATE INDEX lease_payment_schedules_lease_idx
    ON lease_payment_schedules (organization_id, lease_id, installment_number);

INSERT INTO lease_payment_schedules (
    organization_id, lease_id, installment_number, due_day, expected_amount_minor
)
SELECT organization_id, id, 1, payment_day, rent_amount_minor
FROM leases;

ALTER TABLE payments
    ADD COLUMN planned_installment_number smallint
        CHECK (planned_installment_number IS NULL OR planned_installment_number BETWEEN 1 AND 12),
    ADD COLUMN planned_due_date date;

UPDATE payments p
SET planned_installment_number = 1,
    planned_due_date = make_date(
        split_part(c.period, '-', 1)::integer,
        split_part(c.period, '-', 2)::integer,
        l.payment_day
    )
FROM payment_allocations a
JOIN charges c ON c.id = a.charge_id
JOIN leases l ON l.id = c.lease_id
WHERE a.payment_id = p.id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE payments
    DROP COLUMN planned_due_date,
    DROP COLUMN planned_installment_number;
DROP TABLE lease_payment_schedules;
ALTER TABLE leases DROP COLUMN payment_installments;
-- +goose StatementEnd
