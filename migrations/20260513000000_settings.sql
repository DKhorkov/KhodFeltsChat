-- +goose Up
CREATE TABLE IF NOT EXISTS settings
(
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER   NOT NULL UNIQUE,
    theme      INTEGER   NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX settings_user_id_idx ON settings (user_id);

INSERT INTO settings (user_id)
SELECT id FROM users;

-- +goose Down
DROP TABLE IF EXISTS settings;
