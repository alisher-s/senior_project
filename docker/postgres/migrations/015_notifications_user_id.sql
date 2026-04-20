ALTER TABLE notifications_queue ADD COLUMN user_id uuid REFERENCES users(id);

CREATE INDEX notifications_queue_user_id_idx ON notifications_queue (user_id, created_at DESC);
