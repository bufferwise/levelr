-- name: UpsertUserXP :one
INSERT INTO users_xp (guild_id, user_id_str, xp, level, msg_alltime, vc_alltime, created_at, updated_at)
VALUES (sqlc.arg('guildId'), sqlc.arg('userId'), sqlc.arg('xp'), 0, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (guild_id, user_id_str) DO UPDATE SET
    xp = xp + excluded.xp,
    updated_at = CURRENT_TIMESTAMP
RETURNING user_id, guild_id, user_id_str, xp, level, msg_alltime, vc_alltime, last_msg_at, created_at, updated_at;

-- name: IncrementVoiceXP :one
INSERT INTO users_xp (guild_id, user_id_str, xp, level, msg_alltime, vc_alltime, created_at, updated_at)
VALUES (sqlc.arg('guildId'), sqlc.arg('userId'), sqlc.arg('xp'), 0, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (guild_id, user_id_str) DO UPDATE SET
    xp = xp + excluded.xp,
    updated_at = CURRENT_TIMESTAMP
RETURNING user_id, guild_id, user_id_str, xp, level, msg_alltime, vc_alltime, last_msg_at, created_at, updated_at;

-- name: GetUser :one
SELECT user_id, guild_id, user_id_str, xp, level, msg_alltime, vc_alltime, last_msg_at, created_at, updated_at
FROM users_xp
WHERE user_id_str = sqlc.arg('userId') AND guild_id = sqlc.arg('guildId');

-- name: UpdateUserLevel :exec
UPDATE users_xp SET xp = sqlc.arg('xp') WHERE user_id_str = sqlc.arg('userId') AND guild_id = sqlc.arg('guildId');

-- name: GetServerRank :one
SELECT COUNT(*) + 1 AS position FROM users_xp
WHERE (level > sqlc.arg('level')) OR (level = sqlc.arg('level') AND xp > sqlc.arg('xp'));

-- name: GetAllTimeMsgRank :one
SELECT COUNT(*) + 1 AS position FROM users_xp
WHERE msg_alltime > (SELECT ux.msg_alltime FROM users_xp ux WHERE ux.user_id_str = sqlc.arg('userId') AND ux.guild_id = sqlc.arg('guildId'));

-- name: GetAllTimeVCRank :one
SELECT COUNT(*) + 1 AS position FROM users_xp
WHERE vc_alltime > (SELECT ux.vc_alltime FROM users_xp ux WHERE ux.user_id_str = sqlc.arg('userId') AND ux.guild_id = sqlc.arg('guildId'));

-- name: GetAllTimeXPLeaderboard :many
SELECT user_id_str, xp, level FROM users_xp ORDER BY level DESC, xp DESC LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetAllTimeMsgLeaderboard :many
SELECT user_id_str, msg_alltime FROM users_xp ORDER BY msg_alltime DESC LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetAllTimeVCLeaderboard :many
SELECT user_id_str, vc_alltime FROM users_xp ORDER BY vc_alltime DESC LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: SetUserXPAndLevel :one
INSERT INTO users_xp (guild_id, user_id_str, xp, level, msg_alltime, vc_alltime, created_at, updated_at)
VALUES (sqlc.arg('guildId'), sqlc.arg('userId'), sqlc.arg('xp'), sqlc.arg('level'), 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (guild_id, user_id_str) DO UPDATE SET
    xp = excluded.xp,
    level = excluded.level,
    updated_at = CURRENT_TIMESTAMP
RETURNING user_id, guild_id, user_id_str, xp, level, msg_alltime, vc_alltime, last_msg_at, created_at, updated_at;

-- name: AddXPDelta :one
INSERT INTO users_xp (guild_id, user_id_str, xp, level, msg_alltime, vc_alltime, created_at, updated_at)
VALUES (sqlc.arg('guildId'), sqlc.arg('userId'), sqlc.arg('xp'), 0, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (guild_id, user_id_str) DO UPDATE SET
    xp = xp + excluded.xp,
    updated_at = CURRENT_TIMESTAMP
RETURNING user_id, guild_id, user_id_str, xp, level, msg_alltime, vc_alltime, last_msg_at, created_at, updated_at;

-- name: GetTop100Levels :many
SELECT user_id_str, xp, level FROM users_xp ORDER BY level DESC, xp DESC LIMIT 100;

-- name: IncrementMessageCount :exec
INSERT INTO users_xp (guild_id, user_id_str, xp, level, msg_alltime, vc_alltime, created_at, updated_at)
VALUES (sqlc.arg('guildId'), sqlc.arg('userId'), 0, 0, 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (guild_id, user_id_str) DO UPDATE SET
    msg_alltime = msg_alltime + 1,
    updated_at = CURRENT_TIMESTAMP;

-- name: IncrementVoiceMinutes :exec
INSERT INTO users_xp (guild_id, user_id_str, xp, level, msg_alltime, vc_alltime, created_at, updated_at)
VALUES (sqlc.arg('guildId'), sqlc.arg('userId'), 0, 0, 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (guild_id, user_id_str) DO UPDATE SET
    vc_alltime = vc_alltime + 1,
    updated_at = CURRENT_TIMESTAMP;

-- name: GetGuildLeaderboard :many
SELECT user_id_str, xp, level, msg_alltime, vc_alltime
FROM users_xp
WHERE guild_id = sqlc.arg('guildId')
ORDER BY level DESC, xp DESC
LIMIT sqlc.arg('limit');

-- name: GetGuildUserStats :one
SELECT user_id, guild_id, user_id_str, xp, level, msg_alltime, vc_alltime, last_msg_at, created_at, updated_at
FROM users_xp
WHERE user_id_str = sqlc.arg('userId') AND guild_id = sqlc.arg('guildId');

-- name: BatchIncrementCounters :exec
UPDATE users_xp
SET
    msg_alltime = CASE WHEN sqlc.narg('incrementMsg') = 1 THEN msg_alltime + 1 ELSE msg_alltime END,
    vc_alltime = CASE WHEN sqlc.narg('incrementVC') = 1 THEN vc_alltime + 1 ELSE vc_alltime END,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id_str = sqlc.arg('userId') AND guild_id = sqlc.arg('guildId');
