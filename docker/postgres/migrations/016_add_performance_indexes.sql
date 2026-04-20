-- Add missing indexes for common query patterns (performance).
--
-- NOTE: This repo's migration runner (`scripts/apply-migrations.sh`) applies ALL `*.sql`
-- files in this directory in lexicographic order. To avoid accidentally executing a
-- destructive rollback in normal runs, the "down migration" is provided as a comment
-- at the bottom of this file.

-- For GET /tickets/my (filter by user_id)
CREATE INDEX IF NOT EXISTS idx_tickets_user_id ON tickets(user_id);

-- For analytics and ticket count by event
CREATE INDEX IF NOT EXISTS idx_tickets_event_id ON tickets(event_id);

-- For organizer's event list
CREATE INDEX IF NOT EXISTS idx_events_organizer_id ON events(organizer_id);

-- For public event list (only approved events shown)
CREATE INDEX IF NOT EXISTS idx_events_moderation_status ON events(moderation_status);

-- For notifications worker polling
CREATE INDEX IF NOT EXISTS idx_notifications_queue_status ON notifications_queue(status, created_at);

/*
DOWN MIGRATION (manual rollback)
--------------------------------
DROP INDEX IF EXISTS idx_tickets_user_id;
DROP INDEX IF EXISTS idx_tickets_event_id;
DROP INDEX IF EXISTS idx_events_organizer_id;
DROP INDEX IF EXISTS idx_events_moderation_status;
DROP INDEX IF EXISTS idx_notifications_queue_status;
*/

