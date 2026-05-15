-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS web_push_subscriptions
(
    id             SERIAL PRIMARY KEY,
    user_id        INTEGER   NOT NULL,
    endpoint       TEXT      NOT NULL UNIQUE,
    encryption_key TEXT      NOT NULL,
    auth           TEXT      NOT NULL,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX web_push_subscriptions_user_id_idx ON web_push_subscriptions (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS web_push_subscriptions;
-- +goose StatementEnd
