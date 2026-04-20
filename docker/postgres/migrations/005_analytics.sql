-- Analytics module schema (foundation)
CREATE TABLE IF NOT EXISTS analytics_event_stats_snapshots (
  id uuid PRIMARY KEY,
  event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  tickets bigint NOT NULL,
  revenue bigint NOT NULL,
  as_of timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS analytics_event_stats_snapshots_event_id_idx ON analytics_event_stats_snapshots(event_id);
CREATE INDEX IF NOT EXISTS analytics_event_stats_snapshots_as_of_idx ON analytics_event_stats_snapshots(as_of);

