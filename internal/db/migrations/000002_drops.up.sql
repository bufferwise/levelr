-- 000002_drops.up.sql
-- Drop events: tracks every math-question drop and its winner
CREATE TABLE drops (
    id          INTEGER      PRIMARY KEY AUTOINCREMENT,
    question    TEXT         NOT NULL,
    answer      TEXT         NOT NULL,
    xp_amount   INTEGER      NOT NULL,        -- can be negative for admin drops
    winner_id   INTEGER,                       -- NULL until someone answers correctly
    dropped_by  INTEGER,                       -- NULL = auto timer, non-NULL = admin user snowflake
    channel_id  INTEGER      NOT NULL,
    message_id  INTEGER,                       -- the message containing the drop embed
    claimed_at  DATETIME,                      -- when the winner claimed it
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX drops_winner_idx  ON drops (winner_id)  WHERE winner_id IS NOT NULL;
CREATE INDEX drops_pending_idx ON drops (id)          WHERE winner_id IS NULL;
CREATE INDEX drops_time_idx    ON drops (created_at DESC);
