-- Optional cover image URL for events (HTTPS link to hosted image; empty string = none).
ALTER TABLE events
  ADD COLUMN IF NOT EXISTS cover_image_url text NOT NULL DEFAULT '';

