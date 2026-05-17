-- +goose Up
ALTER TABLE settings
    ADD COLUMN email_consents     INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN web_push_consents  INTEGER NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE settings
    DROP COLUMN IF EXISTS email_consents,
    DROP COLUMN IF EXISTS web_push_consents;
