CREATE TABLE users (
    user_id INTEGER PRIMARY KEY,
    guild_id INTEGER NOT NULL,
    user_id_str TEXT NOT NULL,  -- Original Discord ID for reference
    xp INTEGER NOT NULL DEFAULT 0,
    level INTEGER NOT NULL DEFAULT 0,
    msg_alltime INTEGER NOT NULL DEFAULT 0,
    vc_alltime INTEGER NOT NULL DEFAULT 0,
    last_msg_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(guild_id, user_id_str)
);
CREATE TABLE users_old (
    guildId TEXT NOT NULL,
    userId TEXT NOT NULL,
    xp INTEGER DEFAULT 0 NOT NULL,
    PRIMARY KEY (guildId, userId)
);
CREATE TABLE weekly_messages (
    user_id INTEGER NOT NULL,
    week_start TEXT NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, week_start),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);
CREATE TABLE weekly_voice (
    user_id INTEGER NOT NULL,
    week_start TEXT NOT NULL,
    minutes INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, week_start),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);
CREATE TABLE blacklist (
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    added_by INTEGER NOT NULL,
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (entity_type, entity_id)
);
CREATE TABLE blacklists_old (
    guildId TEXT NOT NULL,
    targetId TEXT NOT NULL,
    type TEXT NOT NULL,
    PRIMARY KEY (guildId, targetId)
);
CREATE TABLE multipliers (
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    multiplier REAL NOT NULL DEFAULT 1.0,
    PRIMARY KEY (entity_type, entity_id)
);
CREATE TABLE multipliers_old (
    guildId TEXT NOT NULL,
    targetId TEXT NOT NULL,
    type TEXT NOT NULL,
    multiplier REAL NOT NULL,
    PRIMARY KEY (guildId, targetId)
);
CREATE TABLE bot_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
CREATE TABLE drops (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    xp_amount INTEGER NOT NULL,
    winner_id INTEGER,
    dropped_by INTEGER,
    channel_id INTEGER NOT NULL,
    message_id INTEGER,
    claimed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE sqlite_sequence(name,seq);
CREATE TABLE giveaways (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id INTEGER NOT NULL,
    message_id INTEGER NOT NULL,
    prize TEXT NOT NULL,
    winner_count INTEGER NOT NULL DEFAULT 1,
    required_role INTEGER,
    host_id INTEGER NOT NULL,
    ends_at DATETIME NOT NULL,
    ended BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE giveaway_entries (
    giveaway_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    entered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (giveaway_id, user_id),
    FOREIGN KEY (giveaway_id) REFERENCES giveaways(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);
CREATE INDEX idx_users_guild_id ON users(guild_id);
CREATE INDEX idx_users_xp ON users(xp DESC);
CREATE INDEX idx_users_level ON users(level DESC);
CREATE INDEX idx_users_user_id_str ON users(user_id_str);
CREATE INDEX idx_users_last_msg_at ON users(last_msg_at);
CREATE INDEX idx_weekly_messages_user_week ON weekly_messages(user_id, week_start);
CREATE INDEX idx_weekly_voice_user_week ON weekly_voice(user_id, week_start);
CREATE INDEX idx_drops_channel_id ON drops(channel_id);
CREATE INDEX idx_drops_winner_id ON drops(winner_id);
CREATE INDEX idx_drops_message_id ON drops(message_id);
CREATE INDEX idx_drops_created_at ON drops(created_at);
CREATE INDEX idx_giveaways_channel_id ON giveaways(channel_id);
CREATE INDEX idx_giveaways_message_id ON giveaways(message_id);
CREATE INDEX idx_giveaways_host_id ON giveaways(host_id);
CREATE INDEX idx_giveaways_ends_at ON giveaways(ends_at);
CREATE INDEX idx_giveaways_ended ON giveaways(ended);
CREATE TRIGGER update_users_timestamp 
AFTER UPDATE ON users
BEGIN
    UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE user_id = NEW.user_id;
END;
