-- name: UpsertWeeklyMessage :exec
INSERT INTO weekly_xp (guildId, userId, yearWeek, messages) VALUES (sqlc.arg('guildId'), sqlc.arg('userId'), sqlc.arg('yearWeek'), 1) ON CONFLICT (guildId, userId, yearWeek) DO UPDATE SET messages = messages + 1;

-- name: IncrementWeeklyVoice :exec
INSERT INTO weekly_xp (guildId, userId, yearWeek, voiceMinutes) VALUES (sqlc.arg('guildId'), sqlc.arg('userId'), sqlc.arg('yearWeek'), 1) ON CONFLICT (guildId, userId, yearWeek) DO UPDATE SET voiceMinutes = voiceMinutes + 1;

-- name: GetWeeklyMsgForUser :one
SELECT COALESCE((SELECT messages FROM weekly_xp WHERE guildId = sqlc.arg('guildId') AND userId = sqlc.arg('userId') AND yearWeek = sqlc.arg('yearWeek')), 0) AS msg_count;

-- name: GetWeeklyVCForUser :one
SELECT COALESCE((SELECT voiceMinutes FROM weekly_xp WHERE guildId = sqlc.arg('guildId') AND userId = sqlc.arg('userId') AND yearWeek = sqlc.arg('yearWeek')), 0) AS vc_minutes;

-- name: GetWeeklyMsgRank :one
SELECT COUNT(*) + 1 AS position FROM weekly_xp AS outer_t WHERE outer_t.yearWeek = sqlc.arg('yearWeek') AND outer_t.messages > COALESCE((SELECT messages FROM weekly_xp AS inner_t WHERE inner_t.guildId = sqlc.arg('guildId') AND inner_t.userId = sqlc.arg('userId') AND inner_t.yearWeek = outer_t.yearWeek), 0);

-- name: GetWeeklyVCRank :one
SELECT COUNT(*) + 1 AS position FROM weekly_xp AS outer_t WHERE outer_t.yearWeek = sqlc.arg('yearWeek') AND outer_t.voiceMinutes > COALESCE((SELECT voiceMinutes FROM weekly_xp AS inner_t WHERE inner_t.guildId = sqlc.arg('guildId') AND inner_t.userId = sqlc.arg('userId') AND inner_t.yearWeek = outer_t.yearWeek), 0);

-- name: GetWeeklyMsgLeaderboard :many
SELECT userId, messages as count FROM weekly_xp WHERE guildId = sqlc.arg('guildId') AND yearWeek = sqlc.arg('yearWeek') ORDER BY messages DESC LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetWeeklyVCLeaderboard :many
SELECT userId, voiceMinutes as minutes FROM weekly_xp WHERE guildId = sqlc.arg('guildId') AND yearWeek = sqlc.arg('yearWeek') ORDER BY voiceMinutes DESC LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
