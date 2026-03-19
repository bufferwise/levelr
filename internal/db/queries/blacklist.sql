-- name: AddBlacklist :exec
INSERT INTO blacklist (entity_type, entity_id, added_by) VALUES (sqlc.arg('entityType'), sqlc.arg('targetId'), 0) ON CONFLICT (entity_type, entity_id) DO NOTHING;

-- name: RemoveBlacklist :exec
DELETE FROM blacklist WHERE entity_id = sqlc.arg('entityId');

-- name: IsBlacklisted :one
SELECT EXISTS(
  SELECT 1 FROM blacklist WHERE entity_id = sqlc.arg('entityId') AND entity_type = sqlc.arg('entityType')
) AS is_blacklisted;

-- name: ListBlacklist :many
SELECT entity_type, entity_id, added_by, added_at
FROM blacklist ORDER BY entity_id;
