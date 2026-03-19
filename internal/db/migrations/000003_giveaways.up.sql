-- 000003_giveaways.up.sql
-- Giveaway events: timed prize draws with optional role-gating
CREATE TABLE giveaways (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    channel_id    INTEGER  NOT NULL,
    message_id    INTEGER  NOT NULL,
    prize         TEXT     NOT NULL,
    winner_count  INTEGER  NOT NULL DEFAULT 1,
    required_role INTEGER,                              -- NULL = no role gate
    host_id       INTEGER  NOT NULL,
    ends_at       DATETIME NOT NULL,
    ended         BOOLEAN  NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX gw_active_idx ON giveaways (ended, ends_at) WHERE ended = 0;

-- Each user's entry in a giveaway (composite PK prevents duplicates)
CREATE TABLE giveaway_entries (
    giveaway_id INTEGER NOT NULL REFERENCES giveaways(id),
    user_id     INTEGER NOT NULL,
    entered_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (giveaway_id, user_id)
);
