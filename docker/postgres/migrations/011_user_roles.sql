-- Multi-role support: one row per (user, role) with optional pending approval for organizer.
-- users.role remains for backward compatibility until a later migration drops it.

CREATE TABLE IF NOT EXISTS user_roles (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role text NOT NULL CHECK (role IN ('student', 'organizer', 'admin')),
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('pending', 'active')),
  PRIMARY KEY (user_id, role)
);

CREATE INDEX IF NOT EXISTS user_roles_user_id_idx ON user_roles(user_id);

INSERT INTO user_roles (user_id, role, status)
SELECT id, role, 'active'::text
FROM users
ON CONFLICT (user_id, role) DO NOTHING;
