-- Payments module schema
CREATE TABLE IF NOT EXISTS payments (
  id uuid PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  amount bigint NOT NULL CHECK (amount > 0),
  currency char(3) NOT NULL,
  status text NOT NULL CHECK (status IN ('pending', 'succeeded', 'failed', 'canceled')),
  provider_name text NOT NULL,
  provider_ref text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS payments_user_id_idx ON payments(user_id);
CREATE INDEX IF NOT EXISTS payments_event_id_idx ON payments(event_id);

