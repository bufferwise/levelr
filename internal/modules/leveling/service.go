package leveling

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/snowflake/v2"

	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/services"
)

// XPForLevel returns the target XP required to move FROM current level TO next level.
// Formula: 10 * (level * level) * 10 => 100 * level^2
func XPForLevel(level int) int64 {
	n := int64(level)
	return 100 * n * n
}

// CurrentLevel is no longer used for cumulative XP. Level is tracked explicitly with resets.
func CurrentLevel(xp int64) int {
	return 0 // Removed in favor of state-driven leveling
}

// CheckLevelUp detects a level jump, assigns roles, and triggers notification.
func CheckLevelUp(ctx context.Context, client *bot.Client, guildID, userID snowflake.ID, oldLevel, newLevel int, notifyFn func(ctx context.Context, guildID, userID snowflake.ID, level int)) {
	if newLevel <= oldLevel {
		return
	}

	// Assign roles for every threshold crossed.
	for _, lr := range LevelRoles {
		if oldLevel < lr.Level && newLevel >= lr.Level {
			_ = client.Rest.AddMemberRole(guildID, userID, snowflake.ID(lr.RoleID))
		}
	}

	// Trigger notification
	if notifyFn != nil {
		notifyFn(ctx, guildID, userID, newLevel)
	}
}

// XPService handles all XP-related operations with optimized database access
type XPService struct {
	queries *db.Queries
	client  *bot.Client
	Notify  services.Notifier
}

// GetGuildLeaderboard retrieves optimized leaderboard for a specific guild
func (s *XPService) GetGuildLeaderboard(ctx context.Context, guildID uint64, limit int) ([]db.GetGuildLeaderboardRow, error) {
	return s.queries.GetGuildLeaderboard(ctx, db.GetGuildLeaderboardParams{
		GuildId: int64(guildID),
		Limit:   int64(limit),
	})
}

// GetGuildUserStats retrieves comprehensive user stats for a guild
func (s *XPService) GetGuildUserStats(ctx context.Context, userID, guildID uint64) (db.UsersXp, error) {
	return s.queries.GetGuildUserStats(ctx, db.GetGuildUserStatsParams{
		UserId:  strconv.FormatUint(userID, 10),
		GuildId: int64(guildID),
	})
}

func NewXPService(queries *db.Queries, client *bot.Client, notify services.Notifier) *XPService {
	return &XPService{queries: queries, client: client, Notify: notify}
}

// AwardMessageXP handles the XP award flow for a message.
func (s *XPService) AwardMessageXP(ctx context.Context, userID, guildID uint64, xpToAward int64, weekStart time.Time) error {
	user, err := s.queries.GetUser(ctx, db.GetUserParams{UserId: strconv.FormatUint(userID, 10), GuildId: int64(guildID)})
	if err != nil {
		// New user case - create compatible struct
		user = db.UsersXp{
			Xp:    0,
			Level: 0,
		}
	}

	// Determine target XP for next level
	targetXP := XPForLevel(int(user.Level + 1))
	if targetXP == 0 {
		targetXP = 100
	}

	newLevel := user.Level
	finalXP := user.Xp + xpToAward

	// Level up loop - handles massive XP gains from drops/admin
	levelUpHappened := false
	for finalXP >= targetXP {
		newLevel++
		levelUpHappened = true
		// Re-calculate target for the NEXT level jump
		targetXP = XPForLevel(int(newLevel + 1))
		if targetXP == 0 {
			targetXP = 100
		}
	}

	if levelUpHappened {
		finalXP = 0 // User requested: reset to zero on leveling up
	}

	// Update record
	_, err = s.queries.SetUserXPAndLevel(ctx, db.SetUserXPAndLevelParams{
		GuildId: int64(guildID),
		UserId:  strconv.FormatUint(userID, 10),
		Xp:      finalXP,
		Level:   int64(newLevel),
	})
	if err != nil {
		return err
	}

	// Log message XP to Discord asynchronously to prevent blocking the gateway event loop
	go s.Notify.LogMessageXP(context.Background(), userID, xpToAward, finalXP, int64(newLevel))

	// Always increment message count (using the atomic UPSERT query I just updated)
	_ = s.queries.IncrementMessageCount(ctx, db.IncrementMessageCountParams{
		UserId:  strconv.FormatUint(userID, 10),
		GuildId: int64(guildID),
	})

	// Weekly stats
	err = s.queries.UpsertWeeklyMessage(ctx, db.UpsertWeeklyMessageParams{
		GuildId:  strconv.FormatUint(guildID, 10),
		UserId:   strconv.FormatUint(userID, 10),
		YearWeek: services.WeekStartString(weekStart),
	})
	if err != nil {
		slog.Error("failed to update weekly message count", slog.Uint64("user_id", userID), slog.Any("err", err))
	}

	// Trigger notifications and roles if level changed asynchronously to prevent blocking the gateway event loop
	if levelUpHappened {
		go CheckLevelUp(context.Background(), s.client, snowflake.ID(guildID), snowflake.ID(userID), int(user.Level), int(newLevel), s.Notify.SendLevelUpEmbed)
	}

	return nil
}

// AwardVoiceXP handles the XP award flow for a voice minute.
func (s *XPService) AwardVoiceXP(ctx context.Context, userID, guildID uint64, xpToAward int64, weekStart time.Time) error {
	user, err := s.queries.GetUser(ctx, db.GetUserParams{UserId: strconv.FormatUint(userID, 10), GuildId: int64(guildID)})
	if err != nil {
		user = db.UsersXp{
			Xp:    0,
			Level: 0,
		}
	}
	oldLevel := int(user.Level)

	targetXP := XPForLevel(oldLevel + 1)
	if targetXP == 0 {
		targetXP = 100
	}

	newLevel := user.Level
	newXP := user.Xp + xpToAward

	levelUpHappened := false
	for newXP >= targetXP {
		newLevel++
		levelUpHappened = true
		targetXP = XPForLevel(int(newLevel + 1))
		if targetXP == 0 {
			targetXP = 100
		}
	}

	if levelUpHappened {
		newXP = 0 // User requested: reset to zero
	}

	_, err = s.queries.SetUserXPAndLevel(ctx, db.SetUserXPAndLevelParams{
		GuildId: int64(guildID),
		UserId:  strconv.FormatUint(userID, 10),
		Xp:      newXP,
		Level:   int64(newLevel),
	})
	if err != nil {
		return err
	}

	// Log voice XP to Discord asynchronously to prevent blocking the gateway event loop
	go s.Notify.LogVoiceXP(context.Background(), userID, xpToAward, newXP, int64(newLevel))

	// Increment voice minutes
	_ = s.queries.IncrementVoiceMinutes(ctx, db.IncrementVoiceMinutesParams{
		UserId:  strconv.FormatUint(userID, 10),
		GuildId: int64(guildID),
	})

	// Weekly stats
	err = s.queries.IncrementWeeklyVoice(ctx, db.IncrementWeeklyVoiceParams{
		GuildId:   strconv.FormatUint(guildID, 10),
		UserId:    strconv.FormatUint(userID, 10),
		YearWeek:  services.WeekStartString(weekStart),
	})
	if err != nil {
		slog.Error("failed to update weekly voice minutes", slog.Uint64("user_id", userID), slog.Any("err", err))
	}

	// Role assignment and notification asynchronously to prevent blocking the gateway event loop
	if levelUpHappened {
		go CheckLevelUp(context.Background(), s.client, snowflake.ID(guildID), snowflake.ID(userID), oldLevel, int(newLevel), s.Notify.SendLevelUpEmbed)
	}

	return nil
}
