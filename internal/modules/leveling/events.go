package leveling

import (
	"context"
	"log/slog"
	"math"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"

	appbot "github.com/bufferwise/levelr/internal/bot"
	"github.com/bufferwise/levelr/internal/cache"
	"github.com/bufferwise/levelr/internal/config"
	"github.com/bufferwise/levelr/internal/services"
)

func MessageListener(cfg *config.Config, blSvc *services.BlacklistService, multSvc *services.MultiplierService, xpSvc *XPService) bot.EventListener {
	return bot.NewListenerFunc(func(e *events.MessageCreate) {
		// 1. Guild Filter
		if e.Message.GuildID == nil || !appbot.IsMainGuild(*e.Message.GuildID, cfg) {
			return
		}

		// 2. Ignore bots
		if e.Message.Author.Bot {
			return
		}

		ctx := context.Background()
		userID := uint64(e.Message.Author.ID)
		guildID := uint64(*e.Message.GuildID)
		channelID := uint64(e.Message.ChannelID)

		// 3. Blacklist check
		if e.Message.Member == nil {
			return
		}
		roleIDs := appbot.SnowflakeSliceToUint64(e.Message.Member.RoleIDs)
		guildIDStr := strconv.FormatUint(guildID, 10)
		isBl, _ := blSvc.IsUserBlacklisted(ctx, userID, channelID, roleIDs, guildIDStr)
		if isBl {
			return
		}

		// 4. Award XP
		multiplier, _ := multSvc.Compute(ctx, userID, channelID, roleIDs, guildIDStr)
		xpToAward := int64(math.Round(1.0 * multiplier))
		if xpToAward < 1 {
			xpToAward = 1
		}

		slog.Info("calculating message XP", 
			slog.Uint64("user_id", userID), 
			slog.Float64("multiplier", multiplier), 
			slog.Int64("xp_final", xpToAward))

		weekStart := services.WeekStart(time.Now().UTC())
		err := xpSvc.AwardMessageXP(ctx, userID, guildID, xpToAward, weekStart)
		if err != nil {
			slog.Error("failed to award message xp", slog.Uint64("user_id", userID), slog.Any("err", err))
			return
		}

		slog.Debug("awarded message XP", slog.Uint64("user_id", userID), slog.Int64("xp", xpToAward))
	})
}

func VoiceListener(cfg *config.Config, cacheClient *cache.Client) bot.EventListener {
	return bot.NewListenerFunc(func(e *events.GuildVoiceStateUpdate) {
		// 1. Guild Filter
		if !appbot.IsMainGuild(e.VoiceState.GuildID, cfg) {
			return
		}

		// 2. Ignore bots
		member := e.Member
		if member.User.Bot {
			return
		}

		ctx := context.Background()
		userID := uint64(e.VoiceState.UserID)
		guildID := uint64(e.VoiceState.GuildID)

		// Get AFK channel
		afkChanID := cacheClient.GetAFKChannel(ctx, guildID)

		oldVS := e.OldVoiceState
		newVS := e.VoiceState

		// Decision Tree:
		// JOIN: new is VC, old was nothing/not-VC
		// LEAVE: new is nothing/not-VC, old was VC
		// MOVE: both are VC channels

		isOldInVC := oldVS.ChannelID != nil
		isNewInVC := newVS.ChannelID != nil

		// Case 1: LEAVE (or move to AFK)
		if !isNewInVC || (afkChanID != 0 && uint64(*newVS.ChannelID) == afkChanID) {
			cacheClient.DeleteVoiceSession(ctx, guildID, userID)
			slog.Debug("user left VC", slog.Uint64("user_id", userID))
			return
		}

		// Case 2: JOIN (or move from AFK to VC)
		if isNewInVC && (!isOldInVC || (afkChanID != 0 && uint64(*oldVS.ChannelID) == afkChanID)) {
			cacheClient.SetVoiceSession(ctx, guildID, userID, uint64(*newVS.ChannelID), time.Now().UTC().Unix())
			slog.Debug("user joined VC", slog.Uint64("user_id", userID), slog.Uint64("channel_id", uint64(*newVS.ChannelID)))
			return
		}

		// Case 3: MOVE between channels (not AFK)
		if isNewInVC && isOldInVC && *newVS.ChannelID != *oldVS.ChannelID {
			cacheClient.SetVoiceSession(ctx, guildID, userID, uint64(*newVS.ChannelID), time.Now().UTC().Unix())
			slog.Debug("user moved VC", slog.Uint64("user_id", userID), slog.Uint64("channel_id", uint64(*newVS.ChannelID)))
		}
	})
}
