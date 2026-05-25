-- +goose Up
ALTER TABLE messages_statuses
ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE messages_statuses DROP COLUMN IF EXISTS is_deleted;
