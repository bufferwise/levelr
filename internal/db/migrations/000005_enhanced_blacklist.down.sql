-- 000005_enhanced_blacklist.down.sql

DROP TABLE IF EXISTS blacklist_audit;
DROP TABLE IF EXISTS blacklist;

CREATE TABLE blacklist (
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    added_by INTEGER NOT NULL,
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (entity_type, entity_id)
);
