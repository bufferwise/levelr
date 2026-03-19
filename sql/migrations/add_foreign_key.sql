-- Add foreign key constraints to existing tables
-- This migration adds foreign key relationships to ensure data integrity

-- Add foreign key to drops table if it references users
-- ALTER TABLE drops ADD CONSTRAINT fk_drops_user_id 
-- FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Add foreign key to giveaways table if it references users  
-- ALTER TABLE giveaways ADD CONSTRAINT fk_giveaways_user_id
-- FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Add foreign key to blacklist table if it references users
-- ALTER TABLE blacklist ADD CONSTRAINT fk_blacklist_user_id
-- FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Add any other foreign key constraints as needed
-- Example for multipliers table if it exists
-- ALTER TABLE multipliers ADD CONSTRAINT fk_multipliers_user_id
-- FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;