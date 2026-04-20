-- Notifications module schema (foundation)
CREATE TABLE IF NOT EXISTS notifications_queue (
  id text PRIMARY KEY,
  type text NOT NULL,
  recipient text NOT NULL,
  title text NOT NULL,
  body text NOT NULL,
  status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'processing', 'sent', 'failed')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS notifications_queue_status_idx ON notifications_queue(status);
CREATE INDEX IF NOT EXISTS notifications_queue_created_at_idx ON notifications_queue(created_at);

