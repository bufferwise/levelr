package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bufferwise/levelr/internal/config"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// Notifier defines the interface for level-up notifications and XP log streams.
type Notifier interface {
	SendLevelUpEmbed(ctx context.Context, guildID, userID snowflake.ID, newLevel int)
	SendWeeklyReport(ctx context.Context, guildID snowflake.ID, messageContent string)
	LogMessageXP(ctx context.Context, userID uint64, xpAwarded int64, currentXP int64, currentLevel int64)
	LogVoiceXP(ctx context.Context, userID uint64, xpAwarded int64, currentXP int64, currentLevel int64)
}

const (
	MsgXPLogChannelID   = 1414167119741976676
	VoiceXPLogChannelID = 1414167171705344113
)

type xpLog struct {
	isVoice      bool
	userID       uint64
	xpAwarded    int64
	currentXP    int64
	currentLevel int64
}

type notifier struct {
	client  *bot.Client
	cfg     *config.Config
	logChan chan xpLog
}

// NewNotifier creates a new level-up Notifier and starts the batch log flush worker.
func NewNotifier(client *bot.Client, cfg *config.Config) Notifier {
	n := &notifier{
		client:  client,
		cfg:     cfg,
		logChan: make(chan xpLog, 2000), // large buffer to prevent blocking
	}
	go n.startLogWorker()
	return n
}

func (n *notifier) startLogWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var msgBuffer []string
	var voiceBuffer []string

	flush := func() {
		if len(msgBuffer) > 0 {
			n.sendBulkLog(MsgXPLogChannelID, strings.Join(msgBuffer, "\n"))
			msgBuffer = nil
		}
		if len(voiceBuffer) > 0 {
			n.sendBulkLog(VoiceXPLogChannelID, strings.Join(voiceBuffer, "\n"))
			voiceBuffer = nil
		}
	}

	for {
		select {
		case log, ok := <-n.logChan:
			if !ok {
				flush()
				return
			}
			targetXP := 100 * (log.currentLevel + 1) * (log.currentLevel + 1)
			if log.isVoice {
				line := fmt.Sprintf("🎙️ **XP Awarded** | <@%d> received **+%d XP** for active voice minutes (Level: **%d**, XP: **%d/%d**)", log.userID, log.xpAwarded, log.currentLevel, log.currentXP, targetXP)
				voiceBuffer = append(voiceBuffer, line)
				if len(voiceBuffer) >= 10 {
					n.sendBulkLog(VoiceXPLogChannelID, strings.Join(voiceBuffer, "\n"))
					voiceBuffer = nil
				}
			} else {
				line := fmt.Sprintf("💬 **XP Awarded** | <@%d> received **+%d XP** (Level: **%d**, XP: **%d/%d**)", log.userID, log.xpAwarded, log.currentLevel, log.currentXP, targetXP)
				msgBuffer = append(msgBuffer, line)
				if len(msgBuffer) >= 10 {
					n.sendBulkLog(MsgXPLogChannelID, strings.Join(msgBuffer, "\n"))
					msgBuffer = nil
				}
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (n *notifier) sendBulkLog(channelID uint64, content string) {
	lines := strings.Split(content, "\n")
	var chunk []string
	chunkLen := 0

	sendChunk := func() {
		if len(chunk) == 0 {
			return
		}
		bulkContent := strings.Join(chunk, "\n")
		_, err := (*n.client).Rest.CreateMessage(
			snowflake.ID(channelID),
			discord.NewMessageCreate().
				WithContent(bulkContent).
				WithAllowedMentions(&discord.AllowedMentions{
					Parse: []discord.AllowedMentionType{}, // No pings
				}).
				WithFlags(discord.MessageFlagSuppressNotifications), // Silent message
		)
		if err != nil {
			slog.Error("failed to send bulk XP log to Discord", slog.Uint64("channel_id", channelID), slog.Any("err", err))
		}
		chunk = nil
		chunkLen = 0
	}

	for _, line := range lines {
		if chunkLen+len(line)+1 > 1900 {
			sendChunk()
		}
		chunk = append(chunk, line)
		chunkLen += len(line) + 1
	}
	sendChunk()
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

func (n *notifier) LogMessageXP(ctx context.Context, userID uint64, xpAwarded int64, currentXP int64, currentLevel int64) {
	select {
	case n.logChan <- xpLog{
		isVoice:      false,
		userID:       userID,
		xpAwarded:    xpAwarded,
		currentXP:    currentXP,
		currentLevel: currentLevel,
	}:
	default:
		slog.Warn("XP log buffer full, dropping message XP log", slog.Uint64("user_id", userID))
	}
}

func (n *notifier) LogVoiceXP(ctx context.Context, userID uint64, xpAwarded int64, currentXP int64, currentLevel int64) {
	select {
	case n.logChan <- xpLog{
		isVoice:      true,
		userID:       userID,
		xpAwarded:    xpAwarded,
		currentXP:    currentXP,
		currentLevel: currentLevel,
	}:
	default:
		slog.Warn("XP log buffer full, dropping voice XP log", slog.Uint64("user_id", userID))
	}
}
