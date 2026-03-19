-- 000001_initial.up.sql
-- Core XP table — user_id is Discord snowflake (externally provided BIGINT, NOT SERIAL)
CREATE TABLE users_xp (
    user_id       INTEGER PRIMARY KEY,
    xp            INTEGER     NOT NULL DEFAULT 0,
    level         INTEGER     NOT NULL DEFAULT 0,
    msg_alltime   INTEGER     NOT NULL DEFAULT 0,
    vc_alltime    INTEGER     NOT NULL DEFAULT 0,
    last_msg_at   DATETIME,
    created_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX xp_idx     ON users_xp (xp DESC);
CREATE INDEX msg_at_idx ON users_xp (msg_alltime DESC);
CREATE INDEX vc_at_idx  ON users_xp (vc_alltime DESC);

-- Weekly message stats (one row per user per ISO week)
CREATE TABLE weekly_messages (
    user_id    INTEGER NOT NULL,
    week_start TEXT   NOT NULL,
    count      INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, week_start)
);
CREATE INDEX wmsg_rank_idx ON weekly_messages (week_start, count DESC);

-- Weekly voice stats (one row per user per ISO week)
CREATE TABLE weekly_voice (
    user_id    INTEGER NOT NULL,
    week_start TEXT   NOT NULL,
    minutes    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, week_start)
);
CREATE INDEX wvc_rank_idx ON weekly_voice (week_start, minutes DESC);

-- Blacklist: block users, roles, or channels from earning XP
CREATE TABLE blacklist (
    entity_type TEXT   NOT NULL,
    entity_id   INTEGER NOT NULL,
    added_by    INTEGER NOT NULL,
    added_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (entity_type, entity_id)
);

-- XP multipliers for roles and channels (stackable)
CREATE TABLE multipliers (
    entity_type TEXT         NOT NULL,
    entity_id   INTEGER       NOT NULL,
    multiplier  REAL         NOT NULL DEFAULT 1.0,
    PRIMARY KEY (entity_type, entity_id)
);

-- Bot runtime configuration (key-value)
CREATE TABLE bot_config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT INTO bot_config VALUES ('log_channel_id', '0');
INSERT INTO bot_config VALUES ('msg_xp_cooldown_seconds', '30');
