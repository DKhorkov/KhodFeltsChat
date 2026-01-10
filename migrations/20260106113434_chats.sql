-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS chats
(
    id          SERIAL PRIMARY KEY,
    title       VARCHAR(100),
    description TEXT,
    type        VARCHAR(40) NOT NULL,
    created_at  TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chats_members
(
    id         SERIAL PRIMARY KEY,
    chat_id    INTEGER   NOT NULL,
    user_id    INTEGER   NOT NULL,
    is_read    BOOLEAN   NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chat_id) REFERENCES chats (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX chats_members_chat_id_idx ON chats_members (chat_id);
CREATE INDEX chats_members_user_id_idx ON chats_members (user_id);


CREATE TABLE IF NOT EXISTS messages
(
    id         SERIAL PRIMARY KEY,
    chat_id    INTEGER   NOT NULL,
    sender_id  INTEGER   NOT NULL,
    text       TEXT      NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chat_id) REFERENCES chats (id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX messages_chat_id_idx ON messages (chat_id);
CREATE INDEX messages_created_at_idx ON messages (created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS chats_members;
DROP TABLE IF EXISTS chats;
-- +goose StatementEnd
