-- name: GetConfig :one
SELECT value FROM bot_config WHERE key = ?;

-- name: SetConfig :exec
INSERT INTO bot_config (key, value) VALUES (?, ?) ON CONFLICT (key) DO UPDATE SET value = excluded.value;
