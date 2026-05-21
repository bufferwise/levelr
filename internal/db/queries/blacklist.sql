-- name: AddBlacklist :exec
INSERT INTO blacklist (guild_id, entity_type, entity_id, reason, added_by, is_hidden, expires_at)
VALUES (
    sqlc.arg('guildId'),
    sqlc.arg('entityType'),
    sqlc.arg('entityId'),
    sqlc.arg('reason'),
    sqlc.arg('addedBy'),
    sqlc.arg('isHidden'),
    sqlc.arg('expiresAt')
)
ON CONFLICT (guild_id, entity_type, entity_id) DO UPDATE SET
    reason = excluded.reason,
    added_by = excluded.added_by,
    is_hidden = excluded.is_hidden,
    expires_at = excluded.expires_at,
    added_at = CURRENT_TIMESTAMP;

-- name: RemoveBlacklist :exec
DELETE FROM blacklist 
WHERE guild_id = sqlc.arg('guildId') 
  AND entity_type = sqlc.arg('entityType') 
  AND entity_id = sqlc.arg('entityId');

-- name: GetBlacklistDetails :one
SELECT guild_id, entity_type, entity_id, reason, added_by, is_hidden, expires_at, added_at
FROM blacklist 
WHERE guild_id = sqlc.arg('guildId') 
  AND entity_type = sqlc.arg('entityType') 
  AND entity_id = sqlc.arg('entityId');

-- name: ListGuildBlacklist :many
SELECT guild_id, entity_type, entity_id, reason, added_by, is_hidden, expires_at, added_at
FROM blacklist 
WHERE guild_id = sqlc.arg('guildId')
ORDER BY added_at DESC
LIMIT sqlc.arg('limitVal') OFFSET sqlc.arg('offsetVal');

-- name: PruneExpiredBlacklists :execrows
DELETE FROM blacklist 
WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP;

-- name: AddAuditLog :exec
INSERT INTO blacklist_audit (guild_id, entity_type, entity_id, action, reason, actor_id)
VALUES (
    sqlc.arg('guildId'),
    sqlc.arg('entityType'),
    sqlc.arg('entityId'),
    sqlc.arg('action'),
    sqlc.arg('reason'),
    sqlc.arg('actorId')
);

-- name: GetAuditLogs :many
SELECT id, guild_id, entity_type, entity_id, action, reason, actor_id, created_at
FROM blacklist_audit
WHERE guild_id = sqlc.arg('guildId')
ORDER BY created_at DESC
LIMIT sqlc.arg('limitVal') OFFSET sqlc.arg('offsetVal');
