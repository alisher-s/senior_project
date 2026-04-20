-- Support role-change audit rows (no event) and optional rejection / metadata text.
ALTER TABLE admin_moderation_logs
  ADD COLUMN IF NOT EXISTS reason text;

ALTER TABLE admin_moderation_logs
  ALTER COLUMN event_id DROP NOT NULL;
