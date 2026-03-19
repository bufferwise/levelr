package bot

import (
	"context"
	"log/slog"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"

	"github.com/bufferwise/levelr/internal/config"
)

// SetupBot creates and configures the disgo bot client with required intents and cache options.
func SetupBot(cfg *config.Config, listeners ...bot.EventListener) (*bot.Client, error) {
	client, err := disgo.New(cfg.BotToken,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildMessages,
				gateway.IntentMessageContent, // PRIVILEGED — enable in Discord Dev Portal
				gateway.IntentGuildVoiceStates,
				gateway.IntentGuildMembers, // PRIVILEGED — enable in Discord Dev Portal
			),
		),
		bot.WithCacheConfigOpts(
			cache.WithCaches(
				cache.FlagGuilds,
				cache.FlagMembers,
				cache.FlagVoiceStates,
				cache.FlagChannels,
				cache.FlagRoles,
			),
		),
		bot.WithEventListeners(listeners...),
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// OnReadyFunc returns a listener function for the Ready event.
func OnReadyFunc(cfg *config.Config) func(e *events.Ready) {
	return func(e *events.Ready) {
		slog.Info("bot is ready",
			slog.String("user", e.User.Username),
			slog.Int("guilds", len(e.Guilds)),
		)
	}
}

// VoiceHydrator is the interface for the cache client operations needed during voice hydration.
type VoiceHydrator interface {
	SetAFKChannel(ctx context.Context, guildID, channelID uint64)
	SetVoiceSession(ctx context.Context, guildID, userID, channelID uint64, joinUnix int64)
	GetAFKChannel(ctx context.Context, guildID uint64) uint64
}

// OnGuildsReadyHandler fires after all guilds have been loaded.
// It hydrates active voice sessions into Valkey and caches the AFK channel.
func OnGuildsReadyHandler(cfg *config.Config, cacheClient VoiceHydrator) bot.EventListener {
	return bot.NewListenerFunc(func(e *events.GuildsReady) {
		ctx := context.Background()
		mainGuild := snowflakeFromUint64(cfg.MainGuildID)

		// Cache AFK channel
		if guild, ok := e.Client().Caches.Guild(mainGuild); ok {
			if guild.AfkChannelID != nil {
				cacheClient.SetAFKChannel(ctx, cfg.MainGuildID, uint64(*guild.AfkChannelID))
				slog.Info("cached AFK channel", slog.Uint64("channel_id", uint64(*guild.AfkChannelID)))
			}
		}

		// Hydrate voice sessions from disgo's in-memory cache into Valkey
		var count int
		afkChanID := cacheClient.GetAFKChannel(ctx, cfg.MainGuildID)
		for vs := range e.Client().Caches.VoiceStates(mainGuild) {
			if vs.ChannelID == nil {
				continue
			}
			// Skip bots
			member, ok := e.Client().Caches.Member(mainGuild, vs.UserID)
			if ok && member.User.Bot {
				continue
			}
			// Skip AFK channel
			if afkChanID != 0 && uint64(*vs.ChannelID) == afkChanID {
				continue
			}
			cacheClient.SetVoiceSession(ctx, cfg.MainGuildID, uint64(vs.UserID), uint64(*vs.ChannelID), timeNowUnix())
			count++
		}
		slog.Info("voice hydration complete", slog.Int("sessions", count))
	})
}
