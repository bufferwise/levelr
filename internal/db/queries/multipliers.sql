-- name: SetMultiplier :exec
INSERT INTO multipliers (entity_type, entity_id, multiplier) VALUES (sqlc.arg('entityType'), sqlc.arg('targetId'), sqlc.arg('multiplier')) ON CONFLICT (entity_type, entity_id) DO UPDATE SET multiplier = excluded.multiplier;

-- name: RemoveMultiplier :exec
DELETE FROM multipliers WHERE entity_id = sqlc.arg('entityId');

-- name: GetMultiplier :one
SELECT multiplier FROM multipliers WHERE entity_id = sqlc.arg('entityId') AND entity_type = sqlc.arg('entityType');

-- name: ListMultipliers :many
SELECT entity_type, entity_id, multiplier
FROM multipliers ORDER BY entity_type, multiplier DESC;
