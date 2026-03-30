-- +goose Up
CREATE TABLE IF NOT EXISTS messages_statuses
(
    id         SERIAL PRIMARY KEY,
    message_id INTEGER   NOT NULL,
    user_id    INTEGER   NOT NULL,
    is_read    BOOLEAN   NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX messages_statuses_message_id_idx ON messages_statuses (message_id);
CREATE INDEX messages_statuses_user_id_idx ON messages_statuses (user_id);

-- +goose Down
DROP TABLE IF EXISTS messages_statuses;
