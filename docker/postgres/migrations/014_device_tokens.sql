CREATE TABLE device_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      text NOT NULL,
    platform   text NOT NULL CHECK (platform IN ('android', 'ios')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, token)
);

CREATE INDEX device_tokens_user_id_idx ON device_tokens (user_id);
