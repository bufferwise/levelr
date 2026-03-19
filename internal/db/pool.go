package db

import (
	"context"
	"database/sql"
	"log/slog"

	_ "modernc.org/sqlite"
)

// NewPool creates a SQLite connection and runs migrations.
func NewPool(_ context.Context, databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", databaseURL)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, err
	}

	// Set busy timeout so it waits instead of throwing SQLITE_BUSY
	if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		return nil, err
	}

	// Improve performance with synchronous=NORMAL in WAL mode
	if _, err := db.Exec("PRAGMA synchronous=NORMAL;"); err != nil {
		return nil, err
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Auto-migrate: create tables if they don't exist
	if err := migrate(db); err != nil {
		return nil, err
	}

	slog.Info("sqlite database connected", slog.String("path", databaseURL))
	return db, nil
}

func migrate(db *sql.DB) error {
	migrations := []string{
		// users_xp
		`CREATE TABLE IF NOT EXISTS users_xp (
			user_id       INTEGER PRIMARY KEY,
			xp            INTEGER     NOT NULL DEFAULT 0,
			level         INTEGER     NOT NULL DEFAULT 0,
			msg_alltime   INTEGER     NOT NULL DEFAULT 0,
			vc_alltime    INTEGER     NOT NULL DEFAULT 0,
			last_msg_at   DATETIME,
			created_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS xp_idx     ON users_xp (xp DESC)`,
		`CREATE INDEX IF NOT EXISTS msg_at_idx ON users_xp (msg_alltime DESC)`,
		`CREATE INDEX IF NOT EXISTS vc_at_idx  ON users_xp (vc_alltime DESC)`,

		// weekly_messages
		`CREATE TABLE IF NOT EXISTS weekly_messages (
			user_id    INTEGER NOT NULL,
			week_start TEXT    NOT NULL,
			count      INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (user_id, week_start)
		)`,
		`CREATE INDEX IF NOT EXISTS wmsg_rank_idx ON weekly_messages (week_start, count DESC)`,

		// weekly_voice
		`CREATE TABLE IF NOT EXISTS weekly_voice (
			user_id    INTEGER NOT NULL,
			week_start TEXT    NOT NULL,
			minutes    INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (user_id, week_start)
		)`,
		`CREATE INDEX IF NOT EXISTS wvc_rank_idx ON weekly_voice (week_start, minutes DESC)`,

		// blacklist
		`CREATE TABLE IF NOT EXISTS blacklist (
			entity_type TEXT    NOT NULL,
			entity_id   INTEGER NOT NULL,
			added_by    INTEGER NOT NULL,
			added_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (entity_type, entity_id)
		)`,

		// multipliers
		`CREATE TABLE IF NOT EXISTS multipliers (
			entity_type TEXT    NOT NULL,
			entity_id   INTEGER NOT NULL,
			multiplier  REAL    NOT NULL DEFAULT 1.0,
			PRIMARY KEY (entity_type, entity_id)
		)`,

		// bot_config
		`CREATE TABLE IF NOT EXISTS bot_config (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`INSERT OR IGNORE INTO bot_config VALUES ('log_channel_id', '0')`,
		`INSERT OR IGNORE INTO bot_config VALUES ('msg_xp_cooldown_seconds', '30')`,

		// drops
		`CREATE TABLE IF NOT EXISTS drops (
			id          INTEGER      PRIMARY KEY AUTOINCREMENT,
			question    TEXT         NOT NULL,
			answer      TEXT         NOT NULL,
			xp_amount   INTEGER      NOT NULL,
			winner_id   INTEGER,
			dropped_by  INTEGER,
			channel_id  INTEGER      NOT NULL,
			message_id  INTEGER,
			claimed_at  DATETIME,
			created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS drops_winner_idx  ON drops (winner_id)`,
		`CREATE INDEX IF NOT EXISTS drops_pending_idx ON drops (id) WHERE winner_id IS NULL`,
		`CREATE INDEX IF NOT EXISTS drops_time_idx    ON drops (created_at DESC)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return err
		}
	}
	return nil
}
