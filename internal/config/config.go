package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	BotToken      string
	MainGuildID   uint64
	DBUrl         string
	ValkeyAddr    string
	LogChannelID           uint64
	WeeklyStatsChannelID   uint64
	DropChannelID          uint64 // channel where auto-drops happen
	MsgXPCooldown int    // seconds (0 = no cooldown)
	VCXPAloneDeny bool   // true to exclude alone users

	// Module toggles (all default to true)
	ModuleLeveling bool // MODULE_LEVELING
	ModuleDrops    bool // MODULE_DROPS

	// Drops config (only matters if ModuleDrops=true)
	DropIntervalMin int // DROP_INTERVAL_MINUTES (default: 15)
	DropXPMin       int // DROP_XP_MIN (default: 250)
	DropXPMax       int // DROP_XP_MAX (default: 350)

	// Youtube Presence config
	YoutubePresence bool
	YoutubeHandles  string // comma-separated handles
}

// Load reads .env (if present) and populates a Config.
// Panics on missing required fields to fail fast at startup.
func Load() *Config {
	_ = godotenv.Load() // ignore error — .env is optional in production

	cfg := &Config{
		BotToken:      mustEnv("BOT_TOKEN"),
		MainGuildID:   mustEnvUint64("MAIN_GUILD_ID"),
		DBUrl:         mustEnv("DB_URL"),
		ValkeyAddr:    envOr("VALKEY_ADDR", "localhost:6379"),
		LogChannelID:         envUint64Or("LOG_CHANNEL_ID", 0),
		WeeklyStatsChannelID: envUint64Or("WEEKLY_STATS_CHANNEL_ID", 1412070937280647288),
		DropChannelID:        envUint64Or("DROP_CHANNEL_ID", 0),
		MsgXPCooldown: envIntOr("MSG_XP_COOLDOWN", 0),
		VCXPAloneDeny: envBoolOr("VC_XP_ALONE_DENY", true),
	}

	cfg.ModuleLeveling = envBoolOr("MODULE_LEVELING", true)
	cfg.ModuleDrops = envBoolOr("MODULE_DROPS", true)
	cfg.DropIntervalMin = envIntOr("DROP_INTERVAL_MINUTES", 15)
	cfg.DropXPMin = envIntOr("DROP_XP_MIN", 250)
	cfg.DropXPMax = envIntOr("DROP_XP_MAX", 350)

	cfg.YoutubePresence = envBoolOr("YOUTUBE_PRESENCE", false)
	cfg.YoutubeHandles = envOr("YOUTUBE_HANDLES", "")

	return cfg
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("FATAL: required environment variable %s is not set", key)
	}
	return v
}

func mustEnvUint64(key string) uint64 {
	s := mustEnv(key)
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		log.Fatalf("FATAL: environment variable %s must be a valid uint64, got %q: %v", key, s, err)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envUint64Or(key string, fallback uint64) uint64 {
	s := os.Getenv(key)
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}

func envIntOr(key string, fallback int) int {
	s := os.Getenv(key)
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}

func envBoolOr(key string, fallback bool) bool {
	s := os.Getenv(key)
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return fallback
	}
	return v
}
