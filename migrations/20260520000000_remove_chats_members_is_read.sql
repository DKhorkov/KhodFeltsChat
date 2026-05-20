-- +goose Up
ALTER TABLE chats_members DROP COLUMN is_read;

-- +goose Down
ALTER TABLE chats_members ADD COLUMN is_read BOOLEAN NOT NULL DEFAULT FALSE;
