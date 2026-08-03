-- +goose Up
-- +goose StatementBegin
ALTER TABLE documents
    ADD COLUMN document_date date,
    ADD COLUMN period varchar(7) CHECK (period IS NULL OR period ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    ADD COLUMN issuer text CHECK (issuer IS NULL OR length(trim(issuer)) BETWEEN 1 AND 200),
    ADD COLUMN amount_minor bigint CHECK (amount_minor IS NULL OR amount_minor >= 0),
    ADD COLUMN currency_code varchar(3) CHECK (currency_code IS NULL OR currency_code ~ '^[A-Z]{3}$'),
    ADD COLUMN confidentiality text NOT NULL DEFAULT 'internal'
        CHECK (confidentiality IN ('internal', 'confidential', 'restricted')),
    ADD COLUMN rejection_reason text,
    ADD COLUMN verified_at timestamptz,
    ADD COLUMN verified_by uuid REFERENCES users(id),
    ADD COLUMN updated_by uuid REFERENCES users(id);

CREATE INDEX documents_review_queue_idx
    ON documents (organization_id, status, created_at DESC);
CREATE INDEX documents_document_date_idx
    ON documents (organization_id, document_date DESC)
    WHERE document_date IS NOT NULL;
CREATE INDEX documents_period_idx
    ON documents (organization_id, period)
    WHERE period IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX documents_period_idx;
DROP INDEX documents_document_date_idx;
DROP INDEX documents_review_queue_idx;
ALTER TABLE documents
    DROP COLUMN updated_by,
    DROP COLUMN verified_by,
    DROP COLUMN verified_at,
    DROP COLUMN rejection_reason,
    DROP COLUMN confidentiality,
    DROP COLUMN currency_code,
    DROP COLUMN amount_minor,
    DROP COLUMN issuer,
    DROP COLUMN period,
    DROP COLUMN document_date;
-- +goose StatementEnd
