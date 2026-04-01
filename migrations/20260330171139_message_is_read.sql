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

INSERT INTO messages_statuses (message_id, user_id, is_read)
SELECT messages.id as message_id, users.id as user_id, true
FROM messages
         INNER JOIN chats_members ON messages.chat_id = chats_members.chat_id
         INNER JOIN users ON chats_members.user_id = users.id;

-- +goose Down
DROP TABLE IF EXISTS messages_statuses;
