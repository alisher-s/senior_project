-- Event lifecycle (aligned with ticketing rules in application code).
ALTER TABLE events
  ADD COLUMN IF NOT EXISTS status text NOT NULL DEFAULT 'published'
  CHECK (status IN ('draft', 'published', 'cancelled'));

-- One QR payload hash per ticket (registration retries generate a new random payload).
CREATE UNIQUE INDEX IF NOT EXISTS tickets_qr_hash_hex_uidx ON tickets(qr_hash_hex);
