-- +goose Up
-- +goose StatementBegin
ALTER TABLE organizations
    ADD COLUMN portal_owner_access_code_hash text;

ALTER TABLE tenant_portal_links
    ADD COLUMN tenant_access_code_hash text;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenant_portal_links DROP COLUMN tenant_access_code_hash;
ALTER TABLE organizations DROP COLUMN portal_owner_access_code_hash;
-- +goose StatementEnd
