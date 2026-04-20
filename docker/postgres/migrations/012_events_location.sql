-- Optional venue/location label for events (list-my-tickets and future APIs).
ALTER TABLE events
  ADD COLUMN IF NOT EXISTS location text NOT NULL DEFAULT '';
