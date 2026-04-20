-- Uniqueness per (event, user) only for tickets that still "hold" a seat (active or used).
-- Cancelled rows no longer block a new registration for the same event.
ALTER TABLE tickets DROP CONSTRAINT IF EXISTS tickets_event_id_user_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS tickets_event_user_active_used_uidx
  ON tickets (event_id, user_id)
  WHERE (status IN ('active', 'used'));

