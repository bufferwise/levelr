package bot

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	"github.com/bufferwise/levelr/internal/cache"
	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/handler"
)

var MultiplierSubGroup = discord.ApplicationCommandOptionSubCommandGroup{
	Name:        "multiplier",
	Description: "π Manage XP multipliers",
	Options: []discord.ApplicationCommandOptionSubCommand{
		{
			Name:        "set",
			Description: "Set a multiplier for a role or channel",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "type",
					Description: "Entity type",
					Required:    true,
					Choices: []discord.ApplicationCommandOptionChoiceString{
						{Name: "Role", Value: "role"},
						{Name: "Channel", Value: "channel"},
					},
				},
				discord.ApplicationCommandOptionString{
					Name:        "id",
					Description: "The ID of the entity",
					Required:    true,
				},
				discord.ApplicationCommandOptionString{
					Name:        "value",
					Description: "The multiplier value (e.g. 1.5)",
					Required:    true,
				},
			},
		},
		{
			Name:        "remove",
			Description: "Remove a multiplier for a role or channel",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "type",
					Description: "Entity type",
					Required:    true,
					Choices: []discord.ApplicationCommandOptionChoiceString{
						{Name: "Role", Value: "role"},
						{Name: "Channel", Value: "channel"},
					},
				},
				discord.ApplicationCommandOptionString{
					Name:        "id",
					Description: "The ID of the entity",
					Required:    true,
				},
			},
		},
	},
}

func HandleMultiplierSet(queries *db.Queries, cacheClient *cache.Client) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		data := e.SlashCommandInteractionData()

		eType := data.String("type")
		idStr := data.String("id")
		valStr := data.String("value")
		id, _ := strconv.ParseUint(idStr, 10, 64)
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return handler.RespondEphemeral(e, "Invalid multiplier value provided.")
		}

		err = queries.SetMultiplier(ctx, db.SetMultiplierParams{
			TargetId:   int64(id),
			EntityType: eType,
			Multiplier: val,
		})
		if err != nil {
			return err
		}

		cacheClient.InvalidateMultiplier(ctx, eType, id)

		// Log admin action to Discord log channel
		actorMention := fmt.Sprintf("<@%d>", e.User().ID)
		var targetMention string
		if eType == "channel" {
			targetMention = fmt.Sprintf("<#%d>", id)
		} else {
			targetMention = fmt.Sprintf("<@&%d>", id)
		}
		now := time.Now()
		adminLogEmbed := discord.Embed{
			Title: "π Multiplier Set",
			Description: fmt.Sprintf(
				"**Actor:** %s\n**Target:** %s (`%d`)\n**Type:** %s\n**Value:** **%.2fx**",
				actorMention, targetMention, id, eType, val,
			),
			Color: 0x61AFEF,
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

		return handler.RespondEphemeral(e, fmt.Sprintf("π Set %s `%d` multiplier to **%0.2fx**.", eType, id, val))
	}
}

func HandleMultiplierRemove(queries *db.Queries, cacheClient *cache.Client) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		data := e.SlashCommandInteractionData()
		eType := data.String("type")
		idStr := data.String("id")
		id, _ := strconv.ParseUint(idStr, 10, 64)

		err := queries.RemoveMultiplier(ctx, int64(id))
		if err != nil {
			return err
		}

		cacheClient.InvalidateMultiplier(ctx, eType, id)

		// Log admin action to Discord log channel
		actorMention := fmt.Sprintf("<@%d>", e.User().ID)
		var targetMention string
		if eType == "channel" {
			targetMention = fmt.Sprintf("<#%d>", id)
		} else {
			targetMention = fmt.Sprintf("<@&%d>", id)
		}
		now := time.Now()
		adminLogEmbed := discord.Embed{
			Title: "π Multiplier Removed",
			Description: fmt.Sprintf(
				"**Actor:** %s\n**Target:** %s (`%d`)\n**Type:** %s",
				actorMention, targetMention, id, eType,
			),
			Color: 0xE5C07B,
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

		return handler.RespondEphemeral(e, fmt.Sprintf("π Removed %s `%d` multiplier.", eType, id))
	}
}
