-- ========================================
-- DATABASE: databaseGotBuffed.sql
-- HYPER-ADVANCED MODULAR LEVELING SYSTEM
-- Created for Discord Leveling Bot
-- ========================================

PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = 10000;
PRAGMA temp_store = memory;

-- ========================================
-- CORE USER MANAGEMENT TABLES
-- ========================================

-- Enhanced users table with comprehensive tracking
CREATE TABLE users (
    user_id TEXT PRIMARY KEY,                    -- Discord user ID (string)
    username TEXT NOT NULL,                       -- Discord username (with discriminator)
    display_name TEXT,                            -- Current display name
    avatar_url TEXT,                             -- Avatar URL
    is_bot BOOLEAN DEFAULT FALSE,                -- Bot flag
    is_active BOOLEAN DEFAULT TRUE,              -- Account status
    first_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- User profile extensions
CREATE TABLE user_profiles (
    user_id TEXT PRIMARY KEY,
    bio TEXT,                                    -- User biography
    favorite_color TEXT,                         -- Customization
    banner_url TEXT,                             -- Profile banner
    reputation_score INTEGER DEFAULT 0,           -- Community reputation
    total_commands_used INTEGER DEFAULT 0,        -- Bot interaction count
    preferred_language TEXT DEFAULT 'en',         -- User preference
    timezone_offset INTEGER DEFAULT 0,            -- UTC offset in minutes
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- ========================================
-- ADVANCED LEVELING SYSTEM
-- ========================================

-- Level definitions with rewards and requirements
CREATE TABLE levels (
    level INTEGER PRIMARY KEY,
    level_name TEXT NOT NULL,                    -- e.g., "Beginner", "Expert"
    min_xp_required INTEGER NOT NULL UNIQUE,     -- XP needed to reach this level
    max_xp_range INTEGER NOT NULL,               -- Maximum XP in this level range
    level_color TEXT DEFAULT '#FFFFFF',          -- Display color
    level_icon TEXT,                             -- Icon/emoji representation
    rewards TEXT,                                -- JSON array of rewards
    permissions TEXT,                            -- JSON array of permissions
    is_special_level BOOLEAN DEFAULT FALSE,       -- Special milestone levels
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- User level progression with detailed tracking
CREATE TABLE user_levels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    level INTEGER NOT NULL,
    xp_current INTEGER NOT NULL DEFAULT 0,       -- Current XP in this level
    xp_total INTEGER NOT NULL DEFAULT 0,         -- Total XP accumulated
    level_ups_count INTEGER DEFAULT 0,            -- How many times leveled up
    streak_days INTEGER DEFAULT 0,               -- Consecutive days active
    messages_sent INTEGER DEFAULT 0,             -- Total messages sent
    voice_minutes INTEGER DEFAULT 0,             -- Total voice time in minutes
    achievements_earned TEXT,                     -- JSON array of achievement IDs
    level_up_at DATETIME,                        -- When this level was reached
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (level) REFERENCES levels(level),
    UNIQUE(user_id)  -- One record per user
);

-- ========================================
-- ACTIVITY TRACKING SYSTEM
-- ========================================

-- Message activity tracking
CREATE TABLE message_activity (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    content_length INTEGER NOT NULL,
    word_count INTEGER NOT NULL,
    xp_earned INTEGER DEFAULT 0,
    activity_date DATE NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- Voice activity tracking
CREATE TABLE voice_activity (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    session_start DATETIME NOT NULL,
    session_end DATETIME,
    duration_minutes INTEGER NOT NULL,
    xp_earned INTEGER DEFAULT 0,
    activity_date DATE NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- Daily activity summaries
CREATE TABLE daily_activity_summaries (
    user_id TEXT NOT NULL,
    activity_date DATE NOT NULL,
    messages_count INTEGER DEFAULT 0,
    words_written INTEGER DEFAULT 0,
    voice_minutes INTEGER DEFAULT 0,
    xp_earned INTEGER DEFAULT 0,
    active_channels TEXT,                          -- JSON array of channel IDs
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, activity_date),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- ========================================
-- ACHIEVEMENTS SYSTEM
-- ========================================

-- Achievement definitions
CREATE TABLE achievements (
    id TEXT PRIMARY KEY,                          -- Achievement ID
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    icon TEXT,                                   -- Icon/emoji
    category TEXT NOT NULL,                       -- e.g., "messaging", "voice", "leveling"
    requirement_type TEXT NOT NULL,               -- "messages", "xp", "streak", etc.
    requirement_value INTEGER NOT NULL,
    xp_reward INTEGER DEFAULT 0,
    is_hidden BOOLEAN DEFAULT FALSE,              -- Hidden until unlocked
    is_repeatable BOOLEAN DEFAULT FALSE,          -- Can be earned multiple times
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- User achievements
CREATE TABLE user_achievements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    achievement_id TEXT NOT NULL,
    progress_current INTEGER DEFAULT 0,           -- Current progress
    progress_required INTEGER NOT NULL,           -- Required to complete
    is_completed BOOLEAN DEFAULT FALSE,
    completed_at DATETIME,
    completion_count INTEGER DEFAULT 1,           -- For repeatable achievements
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (achievement_id) REFERENCES achievements(id),
    UNIQUE(user_id, achievement_id)
);

-- ========================================
-- REWARDS SYSTEM
-- ========================================

-- Reward definitions
CREATE TABLE rewards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,                           -- "role", "item", "currency", etc.
    value TEXT NOT NULL,                          -- JSON with reward details
    cost_xp INTEGER NOT NULL DEFAULT 0,
    cost_currency INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    is_limited BOOLEAN DEFAULT FALSE,
    max_claims INTEGER DEFAULT 0,
    current_claims INTEGER DEFAULT 0,
    valid_from DATETIME,
    valid_until DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- User reward inventory
CREATE TABLE user_inventory (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    reward_id INTEGER NOT NULL,
    quantity INTEGER DEFAULT 1,
    acquired_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME,
    is_active BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (reward_id) REFERENCES rewards(id)
);

-- ========================================
-- GUILD-SPECIFIC CONFIGURATIONS
-- ========================================

-- Guild settings
CREATE TABLE guild_settings (
    guild_id TEXT PRIMARY KEY,
    guild_name TEXT,
    xp_multiplier REAL DEFAULT 1.0,
    voice_xp_per_minute INTEGER DEFAULT 1,
    message_xp_per_word INTEGER DEFAULT 1,
    max_daily_xp INTEGER DEFAULT 1000,
    level_up_notifications BOOLEAN DEFAULT TRUE,
    announcement_channel_id TEXT,
    ignored_channels TEXT,                        -- JSON array of channel IDs
    ignored_roles TEXT,                           -- JSON array of role IDs
    custom_level_roles TEXT,                       -- JSON mapping level -> role ID
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Guild member data
CREATE TABLE guild_members (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    nickname TEXT,
    joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_boosting BOOLEAN DEFAULT FALSE,
    boost_start_date DATETIME,
    custom_title TEXT,
    member_notes TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (guild_id, user_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- ========================================
-- XP MULTIPLIERS AND BONUSES
-- ========================================

-- XP multipliers for different contexts
CREATE TABLE xp_multipliers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT,
    user_id TEXT,
    channel_id TEXT,
    role_id TEXT,
    event_type TEXT NOT NULL,                     -- "message", "voice", "achievement", etc.
    multiplier REAL NOT NULL DEFAULT 1.0,
    reason TEXT,                                  -- Why this multiplier exists
    starts_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ends_at DATETIME,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (guild_id) REFERENCES guild_settings(guild_id) ON DELETE CASCADE
);

-- XP bonuses (one-time additions)
CREATE TABLE xp_bonuses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    guild_id TEXT,
    amount INTEGER NOT NULL,
    reason TEXT NOT NULL,
    given_by TEXT,                               -- Admin who gave the bonus
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (guild_id) REFERENCES guild_settings(guild_id) ON DELETE CASCADE
);

-- ========================================
-- ANALYTICS AND REPORTING
-- ========================================

-- Leaderboard snapshots
CREATE TABLE leaderboard_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT NOT NULL,
    snapshot_date DATE NOT NULL,
    top_users TEXT NOT NULL,                      -- JSON array of user rankings
    total_users INTEGER NOT NULL,
    total_xp INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (guild_id) REFERENCES guild_settings(guild_id) ON DELETE CASCADE
);

-- System events log
CREATE TABLE system_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,                     -- "level_up", "achievement", "reward_claim", etc.
    user_id TEXT,
    guild_id TEXT,
    event_data TEXT,                              -- JSON with event details
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (guild_id) REFERENCES guild_settings(guild_id) ON DELETE CASCADE
);

-- ========================================
-- INDEXES FOR PERFORMANCE
-- ========================================

-- User-related indexes
CREATE INDEX idx_users_last_seen ON users(last_seen_at);
CREATE INDEX idx_users_active ON users(is_active);
CREATE INDEX idx_user_profiles_reputation ON user_profiles(reputation_score);

-- Leveling indexes
CREATE INDEX idx_user_levels_user ON user_levels(user_id);
CREATE INDEX idx_user_levels_level ON user_levels(level);
CREATE INDEX idx_user_levels_xp_total ON user_levels(xp_total DESC);
CREATE INDEX idx_levels_min_xp ON levels(min_xp_required);

-- Activity indexes
CREATE INDEX idx_message_activity_user_date ON message_activity(user_id, activity_date);
CREATE INDEX idx_message_activity_guild ON message_activity(guild_id, activity_date);
CREATE INDEX idx_voice_activity_user_date ON voice_activity(user_id, activity_date);
CREATE INDEX idx_daily_activity_user_date ON daily_activity_summaries(user_id, activity_date);

-- Achievement indexes
CREATE INDEX idx_user_achievements_user ON user_achievements(user_id);
CREATE INDEX idx_user_achievements_completed ON user_achievements(is_completed, completed_at);
CREATE INDEX idx_achievements_category ON achievements(category);

-- Guild indexes
CREATE INDEX idx_guild_members_guild ON guild_members(guild_id);
CREATE INDEX idx_guild_members_user ON guild_members(user_id);
CREATE INDEX idx_xp_multipliers_active ON xp_multipliers(is_active, ends_at);

-- Analytics indexes
CREATE INDEX idx_leaderboard_guild_date ON leaderboard_snapshots(guild_id, snapshot_date);
CREATE INDEX idx_system_events_type_date ON system_events(event_type, created_at);
CREATE INDEX idx_system_events_user ON system_events(user_id, created_at);

-- ========================================
-- TRIGGERS FOR AUTOMATIC UPDATES
-- ========================================

-- Update user last_seen_at when any activity occurs
CREATE TRIGGER update_user_last_seen_message
    AFTER INSERT ON message_activity
    BEGIN
        UPDATE users SET last_seen_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP 
        WHERE user_id = NEW.user_id;
    END;

CREATE TRIGGER update_user_last_seen_voice
    AFTER INSERT ON voice_activity
    BEGIN
        UPDATE users SET last_seen_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP 
        WHERE user_id = NEW.user_id;
    END;

-- Update daily activity summary
CREATE TRIGGER update_daily_activity_message
    AFTER INSERT ON message_activity
    BEGIN
        INSERT OR REPLACE INTO daily_activity_summaries 
        (user_id, activity_date, messages_count, words_written, xp_earned, updated_at)
        VALUES (
            NEW.user_id, 
            NEW.activity_date,
            COALESCE((SELECT messages_count FROM daily_activity_summaries 
                     WHERE user_id = NEW.user_id AND activity_date = NEW.activity_date), 0) + 1,
            COALESCE((SELECT words_written FROM daily_activity_summaries 
                     WHERE user_id = NEW.user_id AND activity_date = NEW.activity_date), 0) + NEW.word_count,
            COALESCE((SELECT xp_earned FROM daily_activity_summaries 
                     WHERE user_id = NEW.user_id AND activity_date = NEW.activity_date), 0) + NEW.xp_earned,
            CURRENT_TIMESTAMP
        );
    END;

-- ========================================
-- VIEWS FOR COMMON QUERIES
-- ========================================

-- User ranking view
CREATE VIEW user_rankings AS
SELECT 
    ul.user_id,
    u.username,
    u.display_name,
    ul.level,
    ul.xp_total,
    ul.messages_sent,
    ul.voice_minutes,
    RANK() OVER (ORDER BY ul.xp_total DESC) as global_rank,
    RANK() OVER (ORDER BY ul.level DESC, ul.xp_total DESC) as level_rank
FROM user_levels ul
JOIN users u ON ul.user_id = u.user_id
WHERE u.is_active = TRUE;

-- Guild leaderboard view
CREATE VIEW guild_leaderboard AS
SELECT 
    gm.guild_id,
    gs.guild_name,
    ul.user_id,
    u.username,
    u.display_name,
    ul.level,
    ul.xp_total,
    RANK() OVER (PARTITION BY gm.guild_id ORDER BY ul.xp_total DESC) as guild_rank
FROM guild_members gm
JOIN guild_settings gs ON gm.guild_id = gs.guild_id
JOIN user_levels ul ON gm.user_id = ul.user_id
JOIN users u ON gm.user_id = u.user_id
WHERE u.is_active = TRUE;

-- Achievement progress view
CREATE VIEW achievement_progress AS
SELECT 
    ua.user_id,
    u.username,
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
JOIN users u ON ua.user_id = u.user_id
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

-- ========================================
-- DATA IMPORT FROM LVLS.JSON
-- ========================================

-- Note: This section will be populated with actual data from lvls.json
-- The following is a template for the import process

-- Create temporary staging table for JSON import
CREATE TEMPORARY TABLE temp_level_import (
    level INTEGER,
    user_id TEXT,
    username TEXT
);

-- Import users and their levels from the JSON data
-- This would typically be done via a script that parses the JSON and inserts the data
-- For now, we'll create the structure and the import process can be executed separately

-- Sample import queries (to be executed with actual data):
/*
-- Insert unique users
INSERT OR IGNORE INTO users (user_id, username, display_name)
SELECT DISTINCT 
    user_id, 
    username, 
    username as display_name
FROM temp_level_import;

-- Insert user levels
INSERT OR REPLACE INTO user_levels (user_id, level, xp_current, xp_total)
SELECT 
    tli.user_id,
    tli.level,
    CASE 
        WHEN l.min_xp_required > 0 THEN l.min_xp_required + (FLOOR(RANDOM() * (l.max_xp_range - l.min_xp_required + 1)))
        ELSE 0
    END as xp_current,
    CASE 
        WHEN l.min_xp_required > 0 THEN l.min_xp_required + (FLOOR(RANDOM() * (l.max_xp_range - l.min_xp_required + 1)))
        ELSE 0
    END as xp_total
FROM temp_level_import tli
JOIN levels l ON tli.level = l.level;
*/

-- ========================================
-- DATABASE COMPLETION
-- ========================================

-- Update timestamps
UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE updated_at IS NULL;
UPDATE user_levels SET updated_at = CURRENT_TIMESTAMP WHERE updated_at IS NULL;

-- Vacuum and analyze for optimization
VACUUM;
ANALYZE;

-- Database successfully created with hyper-advanced modular structure
-- Ready for data import from lvls.json
