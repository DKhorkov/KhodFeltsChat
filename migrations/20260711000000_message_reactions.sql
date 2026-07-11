-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS reactions
(
    id         SERIAL PRIMARY KEY,
    emoji      VARCHAR(16) NOT NULL UNIQUE,
    sort_order INTEGER     NOT NULL DEFAULT 0,
    created_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS message_reactions
(
    id          SERIAL PRIMARY KEY,
    message_id  INTEGER   NOT NULL REFERENCES messages (id)  ON DELETE CASCADE,
    user_id     INTEGER   NOT NULL REFERENCES users (id)     ON DELETE CASCADE,
    reaction_id INTEGER   NOT NULL REFERENCES reactions (id) ON DELETE CASCADE,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (message_id, user_id, reaction_id)
);

CREATE INDEX message_reactions_message_id_idx ON message_reactions (message_id);
CREATE INDEX message_reactions_user_id_idx    ON message_reactions (user_id);

INSERT INTO reactions (emoji, sort_order) VALUES
    ('👍', 10),
    ('❤️', 20),
    ('🔥', 30),
    ('💯', 40),
    ('😂', 50),
    ('😮', 60),
    ('😢', 70),
    ('😡', 80);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS message_reactions;
DROP TABLE IF EXISTS reactions;
-- +goose StatementEnd
