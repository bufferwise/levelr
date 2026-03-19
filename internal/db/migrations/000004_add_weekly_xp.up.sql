-- 000004_add_weekly_xp.up.sql
-- Weekly XP tracking table for leaderboards
CREATE TABLE weekly_xp (
    guildId TEXT NOT NULL,
    userId TEXT NOT NULL,
    yearWeek TEXT NOT NULL,
    xp INTEGER DEFAULT 0 NOT NULL,
    messages INTEGER DEFAULT 0 NOT NULL,
    voiceMinutes INTEGER DEFAULT 0 NOT NULL,
    PRIMARY KEY(guildId, userId, yearWeek)
);
CREATE INDEX idx_weekly_xp_idx ON weekly_xp (xp DESC);
