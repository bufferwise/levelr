-- ========================================
-- MIGRATION SCRIPT: Old Database to New Structure
-- ========================================
-- This script migrates data from the existing database 
-- to the new hyper-advanced structure while maintaining compatibility

PRAGMA foreign_keys = OFF; -- Disable during migration for performance

-- Step 1: Backup existing data
CREATE TABLE users_migration_backup AS SELECT * FROM users;
CREATE TABLE weekly_messages_backup AS SELECT * FROM weekly_messages;
CREATE TABLE weekly_voice_backup AS SELECT * FROM weekly_voice;
CREATE TABLE weekly_xp_backup AS SELECT * FROM weekly_xp;
CREATE TABLE blacklist_backup AS SELECT * FROM blacklist;
CREATE TABLE multipliers_backup AS SELECT * FROM multipliers;
CREATE TABLE bot_config_backup AS SELECT * FROM bot_config;
CREATE TABLE drops_backup AS SELECT * FROM drops;
CREATE TABLE giveaways_backup AS SELECT * FROM giveaways;
CREATE TABLE giveaway_entries_backup AS SELECT * FROM giveaway_entries;

-- Step 2: Migrate users to new structure
-- Create user profiles from existing users data
INSERT OR IGNORE INTO user_profiles (user_id_str, username, display_name, first_seen_at, last_seen_at, created_at, updated_at)
SELECT 
    user_id_str,
    user_id_str as username,  -- Use user_id_str as username if no separate username field
    user_id_str as display_name,
    created_at as first_seen_at,
    COALESCE(updated_at, created_at) as last_seen_at,
    created_at,
    COALESCE(updated_at, CURRENT_TIMESTAMP)
FROM users;

-- Step 3: Update existing users table with level calculations
-- Calculate levels based on XP ranges
UPDATE users SET level = CASE 
    WHEN xp >= 50000 THEN 100
    WHEN xp >= 25000 THEN 90
    WHEN xp >= 15000 THEN 80
    WHEN xp >= 10000 THEN 70
    WHEN xp >= 6000 THEN 60
    WHEN xp >= 3000 THEN 50
    WHEN xp >= 1500 THEN 40
    WHEN xp >= 700 THEN 30
    WHEN xp >= 300 THEN 20
    WHEN xp >= 100 THEN 10
    WHEN xp >= 0 THEN 5
    ELSE 0
END;

-- Step 4: Migrate weekly data (already compatible)
-- No changes needed for weekly_messages and weekly_voice tables

-- Step 5: Migrate weekly_xp data (already compatible)
-- No changes needed for weekly_xp table

-- Step 6: Migrate blacklist and multipliers (already compatible)
-- No changes needed for blacklist and multipliers tables

-- Step 7: Migrate bot_config (already compatible)
-- No changes needed for bot_config table

-- Step 8: Migrate drops and giveaways (already compatible)
-- No changes needed for drops, giveaways, and giveaway_entries tables

-- Step 9: Initialize achievement progress for existing users
INSERT OR IGNORE INTO user_achievements (user_id_str, achievement_id, progress_current, progress_required, is_completed)
SELECT 
    u.user_id_str,
    a.id,
    CASE 
        WHEN a.requirement_type = 'messages' THEN u.msg_alltime
        WHEN a.requirement_type = 'level' THEN u.level
        WHEN a.requirement_type = 'xp' THEN u.xp
        ELSE 0
    END as progress_current,
    a.requirement_value,
    CASE 
        WHEN a.requirement_type = 'messages' AND u.msg_alltime >= a.requirement_value THEN 1
        WHEN a.requirement_type = 'level' AND u.level >= a.requirement_value THEN 1
        WHEN a.requirement_type = 'xp' AND u.xp >= a.requirement_value THEN 1
        ELSE 0
    END as is_completed
FROM users u
CROSS JOIN achievements a
WHERE a.requirement_type IN ('messages', 'level', 'xp');

-- Mark completed achievements with completion time
UPDATE user_achievements 
SET completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE is_completed = 1 AND completed_at IS NULL;

-- Step 10: Create initial guild settings (if not exists)
INSERT OR IGNORE INTO guild_settings (guild_id, guild_name, created_at, updated_at)
SELECT 
    DISTINCT CAST(guild_id AS TEXT) as guild_id,
    'Default Guild' as guild_name,
    CURRENT_TIMESTAMP as created_at,
    CURRENT_TIMESTAMP as updated_at
FROM users;

-- Step 11: Create system events for migration
INSERT INTO system_events (event_type, user_id_str, guild_id, event_data, created_at)
SELECT 
    'migration_completed' as event_type,
    u.user_id_str,
    CAST(u.guild_id AS TEXT) as guild_id,
    json_object('original_xp', u.xp, 'calculated_level', u.level, 'messages', u.msg_alltime, 'voice_minutes', u.vc_alltime) as event_data,
    CURRENT_TIMESTAMP as created_at
FROM users u;

-- Step 12: Create leaderboard snapshot for migration
INSERT INTO leaderboard_snapshots (guild_id, snapshot_date, top_users, total_users, total_xp, created_at)
SELECT 
    CAST(guild_id AS TEXT) as guild_id,
    DATE(CURRENT_TIMESTAMP) as snapshot_date,
    json_group_array(json_object('user_id_str', user_id_str, 'xp', xp, 'level', level)) as top_users,
    COUNT(*) as total_users,
    SUM(xp) as total_xp,
    CURRENT_TIMESTAMP as created_at
FROM users
GROUP BY guild_id;

-- Step 13: Re-enable foreign keys
PRAGMA foreign_keys = ON;

-- Step 14: Verify migration
-- Create verification report
SELECT 'Migration Verification Report' as report_title;

SELECT 'Users migrated' as item, COUNT(*) as count FROM users;
SELECT 'User profiles created' as item, COUNT(*) as count FROM user_profiles;
SELECT 'Achievement progress initialized' as item, COUNT(*) as count FROM user_achievements;
SELECT 'Guild settings created' as item, COUNT(*) as count FROM guild_settings;
SELECT 'System events logged' as item, COUNT(*) as count FROM system_events WHERE event_type = 'migration_completed';

-- Step 15: Optimization after migration
ANALYZE;
VACUUM;

-- ========================================
-- POST-MIGRATION VALIDATION QUERIES
-- ========================================

-- Check for any data inconsistencies
SELECT 'Data Consistency Check' as check_title;

-- Users without profiles (should be 0)
SELECT 'Users without profiles' as issue, COUNT(*) as count 
FROM users u 
LEFT JOIN user_profiles p ON u.user_id_str = p.user_id_str 
WHERE p.user_id_str IS NULL;

-- Profiles without users (should be 0)
SELECT 'Profiles without users' as issue, COUNT(*) as count 
FROM user_profiles p 
LEFT JOIN users u ON p.user_id_str = u.user_id_str 
WHERE u.user_id_str IS NULL;

-- Users with negative XP (should be 0)
SELECT 'Users with negative XP' as issue, COUNT(*) as count 
FROM users 
WHERE xp < 0;

-- Users with invalid levels (should be 0)
SELECT 'Users with invalid levels' as issue, COUNT(*) as count 
FROM users 
WHERE level < 0 OR level > 100;

-- ========================================
-- ROLLBACK SCRIPT (if needed)
-- ========================================
/*
-- To rollback the migration, uncomment and run:

-- Restore from backups
DROP TABLE users;
CREATE TABLE users AS SELECT * FROM users_migration_backup;

DROP TABLE weekly_messages;
CREATE TABLE weekly_messages AS SELECT * FROM weekly_messages_backup;

DROP TABLE weekly_voice;
CREATE TABLE weekly_voice AS SELECT * FROM weekly_voice_backup;

DROP TABLE weekly_xp;
CREATE TABLE weekly_xp AS SELECT * FROM weekly_xp_backup;

DROP TABLE blacklist;
CREATE TABLE blacklist AS SELECT * FROM blacklist_backup;

DROP TABLE multipliers;
CREATE TABLE multipliers AS SELECT * FROM multipliers_backup;

DROP TABLE bot_config;
CREATE TABLE bot_config AS SELECT * FROM bot_config_backup;

DROP TABLE drops;
CREATE TABLE drops AS SELECT * FROM drops_backup;

DROP TABLE giveaways;
CREATE TABLE giveaways AS SELECT * FROM giveaways_backup;

DROP TABLE giveaway_entries;
CREATE TABLE giveaway_entries AS SELECT * FROM giveaway_entries_backup;

-- Drop new tables
DROP TABLE user_profiles;
DROP TABLE levels;
DROP TABLE achievements;
DROP TABLE user_achievements;
DROP TABLE guild_settings;
DROP TABLE message_activity;
DROP TABLE voice_activity;
DROP TABLE xp_bonuses;
DROP TABLE xp_multipliers_enhanced;
DROP TABLE leaderboard_snapshots;
DROP TABLE system_events;

-- Drop backup tables
DROP TABLE users_migration_backup;
DROP TABLE weekly_messages_backup;
DROP TABLE weekly_voice_backup;
DROP TABLE weekly_xp_backup;
DROP TABLE blacklist_backup;
DROP TABLE multipliers_backup;
DROP TABLE bot_config_backup;
DROP TABLE drops_backup;
DROP TABLE giveaways_backup;
DROP TABLE giveaway_entries_backup;
*/

-- ========================================
-- MIGRATION COMPLETION
-- ========================================

-- Migration completed successfully
-- All existing data has been preserved and enhanced
-- New features are now available for use
-- Existing queries and code remain compatible
