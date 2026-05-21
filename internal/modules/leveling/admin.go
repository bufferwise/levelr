package leveling

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/handler"
)

var SetLevelSubCommand = discord.ApplicationCommandOptionSubCommand{
	Name:        "setlevel",
	Description: "∑ Set a user's level or XP directly",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionUser{
			Name:        "user",
			Description: "Target user",
			Required:    true,
		},
		discord.ApplicationCommandOptionInt{
			Name:        "level",
			Description: "Target level (auto-calculates XP)",
			Required:    false,
		},
		discord.ApplicationCommandOptionInt{
			Name:        "xp",
			Description: "Target XP (overrides level calculation)",
			Required:    false,
		},
	},
}

func HandleSetLevel(queries *db.Queries) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		data := e.SlashCommandInteractionData()
		target, _ := data.OptUser("user")
		userID := int64(target.ID)

		levelVal, hasLevel := data.OptInt("level")
		xpVal, hasXP := data.OptInt("xp")

		var targetXP int64
		var targetLevel int64

		if hasLevel && hasXP {
			// Case: Both provided - set exactly as requested
			targetLevel = int64(levelVal)
			targetXP = int64(xpVal)
		} else if hasLevel {
			// Case: Level only - set level and reset XP to 0
			targetLevel = int64(levelVal)
			targetXP = 0
		} else if hasXP {
			// Case: XP only - calculate level by "spending" XP
			targetLevel = 0
			targetXP = int64(xpVal)
			for {
				req := XPForLevel(int(targetLevel + 1))
				if req <= 0 {
					req = 100 // Safety default
				}
				if targetXP >= req {
					targetLevel++
					targetXP -= req
				} else {
					break
				}
			}
		} else {
			return handler.RespondEphemeral(e, "∂ Provide either `level` or `xp`.")
		}

		_, err := queries.SetUserXPAndLevel(ctx, db.SetUserXPAndLevelParams{
			GuildId: int64(*e.GuildID()),
			UserId:  strconv.FormatUint(uint64(userID), 10),
			Xp:      targetXP,
			Level:   targetLevel,
		})
		if err != nil {
			return err
		}

		// Log admin action to Discord log channel
		actorMention := fmt.Sprintf("<@%d>", e.User().ID)
		targetMention := fmt.Sprintf("<@%d>", userID)
		now := time.Now()
		adminLogEmbed := discord.Embed{
			Title: "∑ Level/XP Override",
			Description: fmt.Sprintf(
				"**Actor:** %s\n**Target:** %s\n**New Level:** %d\n**New XP:** %d",
				actorMention, targetMention, targetLevel, targetXP,
			),
			Color: 0xC678DD,
			Timestamp: &now,
		}
		go func() {
			_, _ = e.Client().Rest.CreateMessage(snowflake.ID(1414167220262539385), discord.NewMessageCreate().
				WithEmbeds(adminLogEmbed).
				WithAllowedMentions(&discord.AllowedMentions{
					Parse: []discord.AllowedMentionType{}, // No pings
				}).
				WithFlags(discord.MessageFlagSuppressNotifications), // Silent message
			)
		}()

		return e.CreateMessage(discord.NewMessageCreate().WithEmbeds(discord.Embed{
			Title: "∑ Level Override",
			Description: fmt.Sprintf(
				"Set <@%d> to:\n```\nf(x) = Level %d | XP = %d\n```",
				userID, targetLevel, targetXP,
			),
			Color: 0x1a1a2e,
			Footer: &discord.EmbedFooter{
				Text: "f(x) — Function • Admin Override",
			},
		}))
	}
}
