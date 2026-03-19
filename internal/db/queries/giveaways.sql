-- name: InsertGiveaway :one
INSERT INTO giveaways (channel_id, message_id, prize, winner_count, required_role, host_id, ends_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetGiveaway :one
SELECT * FROM giveaways WHERE id = ?;

-- name: GetActiveGiveawayByMessage :one
SELECT * FROM giveaways WHERE message_id = ? AND ended = 0;

-- name: ListExpiredUnended :many
SELECT * FROM giveaways WHERE ended = 0 AND ends_at <= CURRENT_TIMESTAMP;

-- name: EndGiveaway :exec
UPDATE giveaways SET ended = 1 WHERE id = ?;

-- name: UpdateGiveawayMessage :exec
UPDATE giveaways SET message_id = ? WHERE id = ?;

-- name: InsertEntry :exec
INSERT OR IGNORE INTO giveaway_entries (giveaway_id, user_id) VALUES (?, ?);

-- name: CountEntries :one
SELECT COUNT(*) FROM giveaway_entries WHERE giveaway_id = ?;

-- name: ListEntries :many
SELECT user_id FROM giveaway_entries WHERE giveaway_id = ?;

-- name: HasEntry :one
SELECT COUNT(*) > 0 AS entered FROM giveaway_entries
WHERE giveaway_id = ? AND user_id = ?;
