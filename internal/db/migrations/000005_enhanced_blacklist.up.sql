-- 000005_enhanced_blacklist.up.sql

-- 1. Rename existing blacklist table to avoid conflict and keep data
ALTER TABLE blacklist RENAME TO blacklist_old_migration;

-- 2. Create the new precision blacklist table with snowflake strings, hidden flag, expires_at, and reason
CREATE TABLE blacklist (
    guild_id TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT 'Violation of server rules',
    added_by TEXT NOT NULL,
    is_hidden BOOLEAN NOT NULL DEFAULT 0,
    expires_at DATETIME,
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (guild_id, entity_type, entity_id)
);

-- 3. Populate new blacklist table with old data, using default guild ID
INSERT INTO blacklist (guild_id, entity_type, entity_id, added_by, added_at, reason, is_hidden)
SELECT '927811764895227945', entity_type, CAST(entity_id AS TEXT), CAST(added_by AS TEXT), added_at, 'Violation of server rules', 0 FROM blacklist_old_migration;

-- 4. Drop temporary migration table
DROP TABLE blacklist_old_migration;

-- 5. Create new blacklist_audit table
CREATE TABLE blacklist_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    action TEXT NOT NULL, -- 'ADDED', 'REMOVED', 'EXPIRED'
    reason TEXT,
    actor_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
