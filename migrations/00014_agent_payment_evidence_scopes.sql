-- +goose Up
-- +goose StatementBegin
ALTER TABLE api_tokens DROP CONSTRAINT api_tokens_scopes_check;
ALTER TABLE api_tokens
    ADD CONSTRAINT api_tokens_scopes_check CHECK (
        scopes <@ ARRAY[
            'property:read', 'incident:read', 'incident:update', 'rent:read',
            'document:write', 'rent:write'
        ]::text[]
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE api_tokens
SET scopes = array_remove(array_remove(scopes, 'document:write'), 'rent:write')
WHERE scopes && ARRAY['document:write', 'rent:write']::text[];
ALTER TABLE api_tokens DROP CONSTRAINT api_tokens_scopes_check;
ALTER TABLE api_tokens
    ADD CONSTRAINT api_tokens_scopes_check CHECK (
        scopes <@ ARRAY['property:read', 'incident:read', 'incident:update', 'rent:read']::text[]
    );
-- +goose StatementEnd
