-- ========================================
-- DATABASE: databaseGotBuffed_compatible.sql
-- HYPER-ADVANCED MODULAR LEVELING SYSTEM (CODEBASE COMPATIBLE)
-- Created for Discord Leveling Bot
-- Maintains compatibility with existing Go codebase and SQLC
-- ========================================

PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = 10000;
PRAGMA temp_store = memory;

-- ========================================
-- COMPATIBLE CORE TABLES (matching existing codebase)
-- ========================================

-- Main users table - COMPATIBLE with existing User struct
CREATE TABLE users (
    user_id INTEGER PRIMARY KEY,                    -- Auto-increment ID for compatibility
    guild_id INTEGER NOT NULL,                      -- Guild ID (required for existing queries)
    user_id_str TEXT NOT NULL,                       -- Discord user ID as string (from lvls.json)
    xp INTEGER NOT NULL DEFAULT 0,                  -- Current XP
    level INTEGER NOT NULL DEFAULT 0,                -- Current level
    msg_alltime INTEGER NOT NULL DEFAULT 0,          -- Total messages sent
    vc_alltime INTEGER NOT NULL DEFAULT 0,           -- Total voice minutes
    last_msg_at DATETIME,                            -- Last message timestamp
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(guild_id, user_id_str)                    -- Ensure one record per user per guild
);

-- Legacy compatibility tables (kept for migration)
CREATE TABLE users_old (
    guildId TEXT NOT NULL,
    userId TEXT NOT NULL,
    xp INTEGER DEFAULT 0 NOT NULL,
    PRIMARY KEY (guildId, userId)
);

CREATE TABLE blacklists_old (
    guildId TEXT NOT NULL,
    targetId TEXT NOT NULL,
    type TEXT NOT NULL,
    PRIMARY KEY (guildId, targetId)
);

CREATE TABLE multipliers_old (
    guildId TEXT NOT NULL,
    targetId TEXT NOT NULL,
    type TEXT NOT NULL,
    multiplier REAL NOT NULL,
    PRIMARY KEY (guildId, targetId)
);

-- Weekly tracking tables (compatible with existing queries)
CREATE TABLE weekly_messages (
    user_id INTEGER NOT NULL,
    week_start TEXT NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, week_start)
);

CREATE TABLE weekly_voice (
    user_id INTEGER NOT NULL,
    week_start TEXT NOT NULL,
    minutes INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, week_start)
);

CREATE TABLE weekly_xp (
    guildId TEXT NOT NULL,
    userId TEXT NOT NULL,
    yearWeek TEXT NOT NULL,
    xp INTEGER DEFAULT 0 NOT NULL,
    messages INTEGER DEFAULT 0 NOT NULL,
    voiceMinutes INTEGER DEFAULT 0 NOT NULL,
    PRIMARY KEY(guildId, userId, yearWeek)
);

-- Existing system tables (maintained for compatibility)
CREATE TABLE blacklist (
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    added_by INTEGER NOT NULL,
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (entity_type, entity_id)
);

CREATE TABLE multipliers (
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    multiplier REAL NOT NULL DEFAULT 1.0,
    PRIMARY KEY (entity_type, entity_id)
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
    PRIMARY KEY (giveaway_id, user_id)
);

-- ========================================
-- ENHANCED FEATURES (new additions)
-- ========================================

-- Enhanced user profiles
CREATE TABLE user_profiles (
    user_id_str TEXT PRIMARY KEY,                   -- Discord user ID
    username TEXT NOT NULL,                         -- Discord username with discriminator
    display_name TEXT,                              -- Current display name
    avatar_url TEXT,                               -- Avatar URL
    is_bot BOOLEAN DEFAULT FALSE,                   -- Bot flag
    is_active BOOLEAN DEFAULT TRUE,                 -- Account status
    bio TEXT,                                      -- User biography
    favorite_color TEXT DEFAULT '#FFFFFF',         -- Customization
    banner_url TEXT,                               -- Profile banner
    reputation_score INTEGER DEFAULT 0,            -- Community reputation
    total_commands_used INTEGER DEFAULT 0,         -- Bot interaction count
    preferred_language TEXT DEFAULT 'en',          -- User preference
    timezone_offset INTEGER DEFAULT 0,             -- UTC offset in minutes
    first_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id_str) REFERENCES users(user_id_str)
);

-- Level definitions with rewards and requirements
CREATE TABLE levels (
    level INTEGER PRIMARY KEY,
    level_name TEXT NOT NULL,                      -- e.g., "Beginner", "Expert"
    min_xp_required INTEGER NOT NULL UNIQUE,       -- XP needed to reach this level
    max_xp_range INTEGER NOT NULL,                 -- Maximum XP in this level range
    level_color TEXT DEFAULT '#FFFFFF',            -- Display color
    level_icon TEXT,                               -- Icon/emoji representation
    rewards TEXT,                                  -- JSON array of rewards
    permissions TEXT,                              -- JSON array of permissions
    is_special_level BOOLEAN DEFAULT FALSE,        -- Special milestone levels
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Achievement system
CREATE TABLE achievements (
    id TEXT PRIMARY KEY,                           -- Achievement ID
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    icon TEXT,                                     -- Icon/emoji
    category TEXT NOT NULL,                         -- e.g., "messaging", "voice", "leveling"
    requirement_type TEXT NOT NULL,                 -- "messages", "xp", "streak", etc.
    requirement_value INTEGER NOT NULL,
    xp_reward INTEGER DEFAULT 0,
    is_hidden BOOLEAN DEFAULT FALSE,               -- Hidden until unlocked
    is_repeatable BOOLEAN DEFAULT FALSE,           -- Can be earned multiple times
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_achievements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id_str TEXT NOT NULL,
    achievement_id TEXT NOT NULL,
    progress_current INTEGER DEFAULT 0,            -- Current progress
    progress_required INTEGER NOT NULL,            -- Required to complete
    is_completed BOOLEAN DEFAULT FALSE,
    completed_at DATETIME,
    completion_count INTEGER DEFAULT 1,            -- For repeatable achievements
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id_str) REFERENCES user_profiles(user_id_str),
    FOREIGN KEY (achievement_id) REFERENCES achievements(id),
    UNIQUE(user_id_str, achievement_id)
);

-- Guild-specific configurations
CREATE TABLE guild_settings (
    guild_id TEXT PRIMARY KEY,
    guild_name TEXT,
    xp_multiplier REAL DEFAULT 1.0,
    voice_xp_per_minute INTEGER DEFAULT 1,
    message_xp_per_word INTEGER DEFAULT 1,
    max_daily_xp INTEGER DEFAULT 1000,
    level_up_notifications BOOLEAN DEFAULT TRUE,
    announcement_channel_id TEXT,
    ignored_channels TEXT,                          -- JSON array of channel IDs
    ignored_roles TEXT,                             -- JSON array of role IDs
    custom_level_roles TEXT,                        -- JSON mapping level -> role ID
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Activity tracking for analytics
CREATE TABLE message_activity (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id_str TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    content_length INTEGER NOT NULL,
    word_count INTEGER NOT NULL,
    xp_earned INTEGER DEFAULT 0,
    activity_date DATE NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id_str) REFERENCES user_profiles(user_id_str)
);

CREATE TABLE voice_activity (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id_str TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    session_start DATETIME NOT NULL,
    session_end DATETIME,
    duration_minutes INTEGER NOT NULL,
    xp_earned INTEGER DEFAULT 0,
    activity_date DATE NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id_str) REFERENCES user_profiles(user_id_str)
);

-- XP bonuses and multipliers (enhanced)
CREATE TABLE xp_bonuses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id_str TEXT NOT NULL,
    guild_id TEXT,
    amount INTEGER NOT NULL,
    reason TEXT NOT NULL,
    given_by TEXT,                                 -- Admin who gave the bonus
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id_str) REFERENCES user_profiles(user_id_str)
);

CREATE TABLE xp_multipliers_enhanced (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT,
    user_id_str TEXT,
    channel_id TEXT,
    role_id TEXT,
    event_type TEXT NOT NULL,                     -- "message", "voice", "achievement", etc.
    multiplier REAL NOT NULL DEFAULT 1.0,
    reason TEXT,                                  -- Why this multiplier exists
    starts_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ends_at DATETIME,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id_str) REFERENCES user_profiles(user_id_str)
);

-- Analytics and reporting
CREATE TABLE leaderboard_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT NOT NULL,
    snapshot_date DATE NOT NULL,
    top_users TEXT NOT NULL,                      -- JSON array of user rankings
    total_users INTEGER NOT NULL,
    total_xp INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (guild_id) REFERENCES guild_settings(guild_id)
);

CREATE TABLE system_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,                     -- "level_up", "achievement", "reward_claim", etc.
    user_id_str TEXT,
    guild_id TEXT,
    event_data TEXT,                              -- JSON with event details
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id_str) REFERENCES user_profiles(user_id_str)
);

-- ========================================
-- INDEXES FOR PERFORMANCE
-- ========================================

-- Core user indexes (compatible with existing queries)
CREATE INDEX idx_users_xp_desc ON users (xp DESC);
CREATE INDEX idx_users_msg_alltime_desc ON users (msg_alltime DESC);
CREATE INDEX idx_users_vc_alltime_desc ON users (vc_alltime DESC);
CREATE INDEX idx_users_guild_user ON users (guild_id, user_id_str);
CREATE INDEX idx_users_last_msg ON users (last_msg_at DESC);

-- Weekly tracking indexes
CREATE INDEX idx_weekly_messages_rank ON weekly_messages (week_start, count DESC);
CREATE INDEX idx_weekly_voice_rank ON weekly_voice (week_start, minutes DESC);
CREATE INDEX idx_weekly_xp_idx ON weekly_xp (xp DESC);

-- Enhanced feature indexes
CREATE INDEX idx_user_profiles_username ON user_profiles (username);
CREATE INDEX idx_user_profiles_active ON user_profiles (is_active);
CREATE INDEX idx_user_profiles_reputation ON user_profiles (reputation_score DESC);
CREATE INDEX idx_user_profiles_last_seen ON user_profiles (last_seen_at DESC);

CREATE INDEX idx_user_levels_user ON users (user_id_str);
CREATE INDEX idx_user_levels_level ON users (level);
CREATE INDEX idx_user_levels_xp_total ON users (xp DESC);

CREATE INDEX idx_levels_min_xp ON levels (min_xp_required);

CREATE INDEX idx_user_achievements_user ON user_achievements (user_id_str);
CREATE INDEX idx_user_achievements_completed ON user_achievements (is_completed, completed_at);
CREATE INDEX idx_achievements_category ON achievements (category);

CREATE INDEX idx_message_activity_user_date ON message_activity (user_id_str, activity_date);
CREATE INDEX idx_message_activity_guild ON message_activity (guild_id, activity_date);
CREATE INDEX idx_voice_activity_user_date ON voice_activity (user_id_str, activity_date);

CREATE INDEX idx_xp_multipliers_active ON xp_multipliers_enhanced (is_active, ends_at);

CREATE INDEX idx_leaderboard_guild_date ON leaderboard_snapshots (guild_id, snapshot_date);
CREATE INDEX idx_system_events_type_date ON system_events (event_type, created_at);
CREATE INDEX idx_system_events_user ON system_events (user_id_str, created_at);

-- Existing indexes (maintained for compatibility)
CREATE INDEX drops_winner_idx ON drops (winner_id);
CREATE INDEX drops_pending_idx ON drops (id) WHERE winner_id IS NULL;
CREATE INDEX drops_time_idx ON drops (created_at DESC);

-- ========================================
-- TRIGGERS FOR AUTOMATIC UPDATES
-- ========================================

-- Update user last_seen_at when any activity occurs
CREATE TRIGGER update_user_last_seen_message
    AFTER INSERT ON message_activity
    BEGIN
        UPDATE user_profiles SET last_seen_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP 
        WHERE user_id_str = NEW.user_id_str;
    END;

CREATE TRIGGER update_user_last_seen_voice
    AFTER INSERT ON voice_activity
    BEGIN
        UPDATE user_profiles SET last_seen_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP 
        WHERE user_id_str = NEW.user_id_str;
    END;

-- Update main users table when profile changes
CREATE TRIGGER update_users_from_profile
    AFTER UPDATE ON user_profiles
    WHEN NEW.last_seen_at != OLD.last_seen_at
    BEGIN
        UPDATE users SET updated_at = CURRENT_TIMESTAMP 
        WHERE user_id_str = NEW.user_id_str;
    END;

-- ========================================
-- VIEWS FOR COMMON QUERIES
-- ========================================

-- User ranking view (compatible with existing leaderboard queries)
CREATE VIEW user_rankings AS
SELECT 
    u.user_id,
    u.guild_id,
    u.user_id_str,
    u.xp,
    u.level,
    u.msg_alltime,
    u.vc_alltime,
    up.username,
    up.display_name,
    RANK() OVER (ORDER BY u.xp DESC) as global_rank,
    RANK() OVER (PARTITION BY u.guild_id ORDER BY u.xp DESC) as guild_rank
FROM users u
LEFT JOIN user_profiles up ON u.user_id_str = up.user_id_str
WHERE up.is_active = TRUE OR up.is_active IS NULL;

-- Guild leaderboard view
CREATE VIEW guild_leaderboard AS
SELECT 
    u.guild_id,
    gs.guild_name,
    u.user_id_str,
    up.username,
    up.display_name,
    u.level,
    u.xp,
    u.msg_alltime,
    u.vc_alltime,
    RANK() OVER (PARTITION BY u.guild_id ORDER BY u.xp DESC) as guild_rank
FROM users u
LEFT JOIN user_profiles up ON u.user_id_str = up.user_id_str
LEFT JOIN guild_settings gs ON CAST(u.guild_id AS TEXT) = gs.guild_id
WHERE up.is_active = TRUE OR up.is_active IS NULL;

-- Achievement progress view
CREATE VIEW achievement_progress AS
SELECT 
    ua.user_id_str,
    up.username,
    a.id as achievement_id,
    a.name,
    a.description,
    a.category,
    ua.progress_current,
    ua.progress_required,
    ua.is_completed,
    ua.completed_at,
    ROUND((ua.progress_current * 100.0 / ua.progress_required), 2) as progress_percentage
FROM user_achievements ua
JOIN user_profiles up ON ua.user_id_str = up.user_id_str
JOIN achievements a ON ua.achievement_id = a.id
WHERE ua.is_completed = FALSE OR ua.is_completed IS NULL;

-- ========================================
-- INITIAL DATA SETUP
-- ========================================

-- Insert level definitions based on the JSON structure
INSERT INTO levels (level, level_name, min_xp_required, max_xp_range, level_color, is_special_level) VALUES
(5, 'Novice', 0, 99, '#95A5A6', FALSE),
(10, 'Apprentice', 100, 299, '#3498DB', FALSE),
(20, 'Journeyman', 300, 699, '#2ECC71', FALSE),
(30, 'Expert', 700, 1499, '#F39C12', FALSE),
(40, 'Master', 1500, 2999, '#E74C3C', TRUE),
(50, 'Grandmaster', 3000, 5999, '#9B59B6', TRUE),
(60, 'Legend', 6000, 9999, '#1ABC9C', TRUE),
(70, 'Mythic', 10000, 14999, '#E67E22', TRUE),
(80, 'Eternal', 15000, 24999, '#34495E', TRUE),
(90, 'Divine', 25000, 49999, '#F1C40F', TRUE),
(100, 'Transcendent', 50000, 999999, '#E91E63', TRUE);

-- Insert basic achievements
INSERT INTO achievements (id, name, description, category, requirement_type, requirement_value, xp_reward) VALUES
('first_message', 'First Steps', 'Send your first message', 'messaging', 'messages', 1, 10),
('message_100', 'Chatterbox', 'Send 100 messages', 'messaging', 'messages', 100, 50),
('message_1000', 'Conversation Master', 'Send 1000 messages', 'messaging', 'messages', 1000, 200),
('voice_60', 'Voice Activated', 'Spend 60 minutes in voice channels', 'voice', 'voice_minutes', 60, 30),
('voice_600', 'Voice Veteran', 'Spend 600 minutes in voice channels', 'voice', 'voice_minutes', 600, 150),
('level_10', 'Rising Star', 'Reach level 10', 'leveling', 'level', 10, 100),
('level_50', 'Elite Status', 'Reach level 50', 'leveling', 'level', 50, 500),
('streak_7', 'Week Warrior', 'Maintain a 7-day activity streak', 'streak', 'streak_days', 7, 75),
('streak_30', 'Monthly Champion', 'Maintain a 30-day activity streak', 'streak', 'streak_days', 30, 300);

-- Insert bot config (existing defaults)
INSERT OR IGNORE INTO bot_config VALUES ('log_channel_id', '0');
INSERT OR IGNORE INTO bot_config VALUES ('msg_xp_cooldown_seconds', '30');

-- ========================================
-- MIGRATION COMPATIBILITY
-- ========================================

-- Create migration helper view for old queries
CREATE VIEW old_users_compatible AS
SELECT 
    CAST(guild_id AS TEXT) as guildId,
    user_id_str as userId,
    xp
FROM users;

-- ========================================
-- DATABASE COMPLETION
-- ========================================

-- Update timestamps
UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE updated_at IS NULL;
UPDATE user_profiles SET updated_at = CURRENT_TIMESTAMP WHERE updated_at IS NULL;

-- Vacuum and analyze for optimization
VACUUM;
ANALYZE;

-- Database successfully created with hyper-advanced modular structure
-- Compatible with existing Go codebase and SQLC
-- Ready for data import from lvls.json and migration from old database
