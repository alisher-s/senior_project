-- Dev seed events: use MinIO public URLs (objects seeded on API startup from static/).
-- Idempotent: only rewrites the legacy /api/v1/static/ prefix.
UPDATE events
SET cover_image_url = REPLACE(
  cover_image_url,
  'http://localhost:8080/api/v1/static/',
  'http://localhost:9000/event-assets/'
)
WHERE cover_image_url LIKE 'http://localhost:8080/api/v1/static/posters/%';
