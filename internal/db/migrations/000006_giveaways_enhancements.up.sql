-- 000006_giveaways_enhancements.up.sql
-- Expand giveaways with optional advanced level requirements, account age checks, and role multiplier mapping
ALTER TABLE giveaways ADD COLUMN min_level INTEGER NOT NULL DEFAULT 0;
ALTER TABLE giveaways ADD COLUMN min_account_days INTEGER NOT NULL DEFAULT 0;
ALTER TABLE giveaways ADD COLUMN role_multipliers TEXT NOT NULL DEFAULT '{}';
