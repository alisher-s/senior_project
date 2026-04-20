-- Admin module schema (foundation)
CREATE TABLE IF NOT EXISTS admin_moderation_logs (
  id uuid PRIMARY KEY,
  admin_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  action text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS admin_moderation_logs_event_id_idx ON admin_moderation_logs(event_id);

