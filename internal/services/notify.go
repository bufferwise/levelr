package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bufferwise/levelr/internal/config"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// Notifier defines the interface for level-up notifications.
type Notifier interface {
	SendLevelUpEmbed(ctx context.Context, guildID, userID snowflake.ID, newLevel int)
	SendWeeklyReport(ctx context.Context, guildID snowflake.ID, messageContent string)
}

type notifier struct {
	client *bot.Client
	cfg    *config.Config
}

// NewNotifier creates a new level-up Notifier.
func NewNotifier(client *bot.Client, cfg *config.Config) Notifier {
	return &notifier{client: client, cfg: cfg}
}

func (n *notifier) SendLevelUpEmbed(ctx context.Context, guildID, userID snowflake.ID, newLevel int) {
	content := fmt.Sprintf(
		"> <:globe:1347988344948260884> ** <@%d> Congratulations! You Just Reached Level %d **\n\n> <:Spider_Sparkle:1336641144674717696>   **Keep Grinding To Obtain More Perks By Leveling Up In <#%d> And To Be In List Of Active Spiders!**",
		userID, newLevel, n.cfg.DropChannelID,
	)

	channelID := snowflake.ID(n.cfg.WeeklyStatsChannelID)

	_, err := (*n.client).Rest.CreateMessage(
		channelID,
		discord.NewMessageCreate().WithContent(content),
	)
	if err != nil {
		slog.Error("failed to send level up message", slog.Any("err", err))
	} else {
		slog.Info("level up message sent", slog.Uint64("user_id", uint64(userID)), slog.Int("level", newLevel))
	}
}

func (n *notifier) SendWeeklyReport(ctx context.Context, guildID snowflake.ID, messageContent string) {
	channelID := snowflake.ID(n.cfg.WeeklyStatsChannelID)

	_, err := (*n.client).Rest.CreateMessage(
		channelID,
		discord.NewMessageCreate().WithContent(messageContent),
	)
	if err != nil {
		slog.Error("failed to send weekly report message", slog.Any("err", err))
	} else {
		slog.Info("weekly report message sent")
	}
}
