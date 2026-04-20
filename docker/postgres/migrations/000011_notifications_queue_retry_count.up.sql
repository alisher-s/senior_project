-- Track SMTP send retries for notifications_queue
ALTER TABLE notifications_queue
  ADD COLUMN IF NOT EXISTS retry_count integer NOT NULL DEFAULT 0;

