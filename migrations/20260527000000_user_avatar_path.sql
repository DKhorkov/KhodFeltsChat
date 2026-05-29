-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN avatar_path TEXT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN IF EXISTS avatar_path;
-- +goose StatementEnd
