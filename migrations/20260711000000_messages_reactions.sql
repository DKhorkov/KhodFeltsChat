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

CREATE TABLE IF NOT EXISTS messages_reactions
(
    id          SERIAL PRIMARY KEY,
    message_id  INTEGER   NOT NULL REFERENCES messages (id)  ON DELETE CASCADE,
    user_id     INTEGER   NOT NULL REFERENCES users (id)     ON DELETE CASCADE,
    reaction_id INTEGER   NOT NULL REFERENCES reactions (id) ON DELETE CASCADE,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (message_id, user_id, reaction_id)
);

CREATE INDEX messages_reactions_message_id_idx ON messages_reactions (message_id);
CREATE INDEX messages_reactions_user_id_idx    ON messages_reactions (user_id);

INSERT INTO reactions (emoji, sort_order) VALUES
    ('👍', 10),
    ('❤️', 20),
    ('🔥', 30),
    ('😁', 40),
    ('😅', 50),
    ('💯', 60),
    ('👀', 70),
    ('😂', 80),
    ('🥰', 90),
    ('🤝', 100),
    ('😮', 110),
    ('😢', 120),
    ('🫡', 130),
    ('😡', 140),
    ('😭', 150),
    ('😈', 160),
    ('😌', 170),
    ('😉', 180),
    ('😏', 190),
    ('🤔', 200),
    ('🥹', 210),
    ('🫣', 220),
    ('🤡', 230),
    ('💩', 240);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS messages_reactions_message_id_idx;
DROP INDEX IF EXISTS messages_reactions_user_id_idx;
DROP TABLE IF EXISTS messages_reactions;
DROP TABLE IF EXISTS reactions;
-- +goose StatementEnd
