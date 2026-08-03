-- +goose Up
ALTER TABLE document_uploads
    ADD COLUMN multipart_upload_id text;

-- +goose Down
ALTER TABLE document_uploads
    DROP COLUMN multipart_upload_id;
