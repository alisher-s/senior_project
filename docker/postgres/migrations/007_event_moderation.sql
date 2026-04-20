-- Admin moderation workflow for events.
-- Note: lifecycle remains in `events.status` (draft/published/cancelled). Moderation uses `moderation_status`
-- because a second column named `status` would collide with the existing lifecycle column.
ALTER TABLE events
  ADD COLUMN IF NOT EXISTS moderation_status text NOT NULL DEFAULT 'approved'
    CHECK (moderation_status IN ('pending', 'approved', 'rejected'));

-- New events default to pending; existing rows keep approved from the ADD COLUMN default above.
ALTER TABLE events ALTER COLUMN moderation_status SET DEFAULT 'pending';

ALTER TABLE events
  ADD COLUMN IF NOT EXISTS moderated_by uuid NULL REFERENCES users(id);

ALTER TABLE events
  ADD COLUMN IF NOT EXISTS organizer_id uuid NULL REFERENCES users(id);

CREATE INDEX IF NOT EXISTS events_moderation_status_idx ON events(moderation_status);
