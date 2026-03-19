package leveling

import (
	"context"
	"log/slog"
	"math"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/snowflake/v2"

	appbot "github.com/bufferwise/levelr/internal/bot"
	"github.com/bufferwise/levelr/internal/cache"
	"github.com/bufferwise/levelr/internal/config"
	"github.com/bufferwise/levelr/internal/services"
)

func StartVoiceTicker(cfg *config.Config, client *bot.Client, cacheClient *cache.Client, blSvc *services.BlacklistService, multSvc *services.MultiplierService, xpSvc *XPService) func(ctx context.Context) {
	return func(ctx context.Context) {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		slog.Info("voice heartbeat ticker started [leveling module]")

		for {
			select {
			case <-ctx.Done():
				slog.Info("voice heartbeat ticker stopped")
				return
			case <-ticker.C:
				processHeartbeat(ctx, cfg, client, cacheClient, blSvc, multSvc, xpSvc)
			}
		}
	}
}

func processHeartbeat(ctx context.Context, cfg *config.Config, client *bot.Client, cacheClient *cache.Client, blSvc *services.BlacklistService, multSvc *services.MultiplierService, xpSvc *XPService) {
	// 1. Scan active sessions from Valkey
	sessions, err := cacheClient.GetActiveVoiceSessions(ctx, cfg.MainGuildID)
	if err != nil {
		slog.Error("failed to scan active voice sessions", slog.Any("err", err))
		return
	}

	if len(sessions) == 0 {
		return
	}

	mainGuildID := snowflake.ID(cfg.MainGuildID)
	weekStart := services.WeekStart(time.Now().UTC())

	for _, session := range sessions {
		userID := snowflake.ID(session.UserID)

		// 2. State verification via Discord cache
		vs, ok := client.Caches.VoiceState(mainGuildID, userID)
		if !ok || vs.ChannelID == nil {
			// User left VC but Valkey stale — cleanup
			cacheClient.DeleteVoiceSession(ctx, cfg.MainGuildID, session.UserID)
			continue
		}

		// 3. Exclusion checks
		// - Server deafened
		if vs.GuildDeaf || vs.SelfDeaf {
			continue
		}

		member, ok := client.Caches.Member(mainGuildID, userID)
		if !ok || member.User.Bot {
			continue
		}

		// - Alone Check (if configured)
		if cfg.VCXPAloneDeny {
			channelMembers := 0
			for ovs := range client.Caches.VoiceStates(mainGuildID) {
				if ovs.ChannelID != nil && *ovs.ChannelID == *vs.ChannelID {
					// Count non-bot members
					if m, ok := client.Caches.Member(mainGuildID, ovs.UserID); ok && !m.User.Bot {
						channelMembers++
					}
				}
			}
			if channelMembers < 2 {
				continue
			}
		}

		// 4. Blacklist check
		roleIDs := appbot.SnowflakeSliceToUint64(member.RoleIDs)
		guildIDStr := strconv.FormatUint(uint64(mainGuildID), 10)
		isBl, _ := blSvc.IsUserBlacklisted(ctx, uint64(userID), uint64(*vs.ChannelID), roleIDs, guildIDStr)
		if isBl {
			slog.Debug("skipping voice XP (blacklisted)", slog.Uint64("user_id", uint64(userID)))
			continue
		}

		// 5. Award XP
		multiplier, _ := multSvc.Compute(ctx, uint64(userID), uint64(*vs.ChannelID), roleIDs, strconv.FormatUint(uint64(mainGuildID), 10))
		xpToAward := int64(math.Round(10.0 * multiplier)) // 10 base XP
		if xpToAward < 1 {
			xpToAward = 1
		}

		slog.Info("calculating voice XP", 
			slog.Uint64("user_id", uint64(userID)), 
			slog.Float64("multiplier", multiplier), 
			slog.Int64("xp_final", xpToAward))

		err := xpSvc.AwardVoiceXP(ctx, uint64(userID), uint64(mainGuildID), xpToAward, weekStart)
		if err != nil {
			slog.Error("failed to award voice XP", slog.Uint64("user_id", uint64(userID)), slog.Any("err", err))
		}
	}
}
