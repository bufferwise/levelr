-- Fix foreign key constraints
-- This migration fixes any issues with existing foreign key constraints

-- First, drop any existing problematic foreign keys
-- ALTER TABLE drops DROP CONSTRAINT IF EXISTS fk_drops_user_id;
-- ALTER TABLE giveaways DROP CONSTRAINT IF EXISTS fk_giveaways_user_id;
-- ALTER TABLE blacklist DROP CONSTRAINT IF EXISTS fk_blacklist_user_id;

-- Re-add foreign keys with proper constraints
-- ALTER TABLE drops ADD CONSTRAINT fk_drops_user_id 
-- FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- ALTER TABLE giveaways ADD CONSTRAINT fk_giveaways_user_id
-- FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- ALTER TABLE blacklist ADD CONSTRAINT fk_blacklist_user_id
-- FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Verify foreign key integrity
-- PRAGMA foreign_key_check;