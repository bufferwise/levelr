package bot

import (
	"context"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/bufferwise/levelr/internal/cache"
	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/handler"
)

var BlacklistSubGroup = discord.ApplicationCommandOptionSubCommandGroup{
	Name:        "blacklist",
	Description: "∫ Manage the XP blacklist",
	Options: []discord.ApplicationCommandOptionSubCommand{
		{
			Name:        "add",
			Description: "Add an entity to the blacklist",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "type",
					Description: "Entity type",
					Required:    true,
					Choices: []discord.ApplicationCommandOptionChoiceString{
						{Name: "User", Value: "user"},
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
		{
			Name:        "remove",
			Description: "Remove an entity from the blacklist",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "type",
					Description: "Entity type",
					Required:    true,
					Choices: []discord.ApplicationCommandOptionChoiceString{
						{Name: "User", Value: "user"},
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

func HandleBlacklistAdd(queries *db.Queries, cacheClient *cache.Client) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		data := e.SlashCommandInteractionData()
		eType := data.String("type")
		idStr := data.String("id")
		id, _ := strconv.ParseUint(idStr, 10, 64)

		err := queries.AddBlacklist(ctx, db.AddBlacklistParams{
			TargetId:   int64(id),
			EntityType: eType,
		})
		if err != nil {
			return err
		}
		cacheClient.InvalidateBlacklist(ctx, eType, id)
		return handler.RespondEphemeral(e, "∫ Added "+eType+" `"+idStr+"` to blacklist.")
	}
}

func HandleBlacklistRemove(queries *db.Queries, cacheClient *cache.Client) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		data := e.SlashCommandInteractionData()
		eType := data.String("type")
		idStr := data.String("id")
		id, _ := strconv.ParseUint(idStr, 10, 64)

		err := queries.RemoveBlacklist(ctx, int64(id))
		if err != nil {
			return err
		}
		cacheClient.InvalidateBlacklist(ctx, eType, id)
		return handler.RespondEphemeral(e, "∫ Removed "+eType+" `"+idStr+"` from blacklist.")
	}
}
