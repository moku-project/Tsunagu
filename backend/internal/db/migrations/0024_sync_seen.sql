-- What each device last said it holds, relayed between devices, so deletion records every device has seen can go.
CREATE TABLE sync_seen (
    node TEXT PRIMARY KEY,
    vv TEXT NOT NULL DEFAULT '{}',
    reported_at INTEGER NOT NULL DEFAULT 0
);
