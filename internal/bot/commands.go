package bot

import (
	"log/slog"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// RegisterCommands registers all slash commands with Discord for the main guild,
// and explicitly wipes any lingering global commands to prevent duplicates.
func RegisterCommands(client *bot.Client, guildID uint64, commands ...discord.ApplicationCommandCreate) error {
	// 1. Wipe global commands so old cached ones vanish across all servers
	_, err := client.Rest.SetGlobalCommands(client.ApplicationID, []discord.ApplicationCommandCreate{})
	if err != nil {
		slog.Warn("failed to wipe global commands", slog.Any("err", err))
	}

	// 2. Strictly overwrite the active guild with our exact local command array
	_, err = client.Rest.SetGuildCommands(client.ApplicationID, snowflake.ID(guildID), commands)
	if err != nil {
		return err
	}
	slog.Info("registered guild commands (wiped globals)", slog.Uint64("guild_id", guildID), slog.Int("count", len(commands)))
	return nil
}
