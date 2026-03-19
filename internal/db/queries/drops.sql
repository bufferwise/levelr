-- name: InsertDrop :one
INSERT INTO drops (question, answer, xp_amount, dropped_by, channel_id, message_id)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ClaimDrop :one
UPDATE drops
SET winner_id = ?, claimed_at = CURRENT_TIMESTAMP
WHERE id = ? AND winner_id IS NULL
RETURNING *;

-- name: GetDrop :one
SELECT * FROM drops WHERE id = ?;

-- name: GetPendingDropByMessage :one
SELECT * FROM drops WHERE message_id = ? AND winner_id IS NULL;
