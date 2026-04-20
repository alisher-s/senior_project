-- Optional event end time (for ticket expiry and check-in window when set).
ALTER TABLE events
  ADD COLUMN IF NOT EXISTS end_at timestamptz NULL;

