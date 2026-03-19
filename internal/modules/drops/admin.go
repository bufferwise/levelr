package drops

import (
	"context"
	"fmt"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/handler"
)

var DropSubCommand = discord.ApplicationCommandOptionSubCommand{
	Name:        "drop",
	Description: "Δ Manually trigger an XP drop or award/deduct XP",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionUser{
			Name:        "user",
			Description: "Target user (for direct XP award/deduction)",
			Required:    false,
		},
		discord.ApplicationCommandOptionInt{
			Name:        "amount",
			Description: "XP amount (negative to deduct)",
			Required:    false,
		},
	},
}

func HandleAdminDrop(queries *db.Queries, dropSvc *DropService) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		data := e.SlashCommandInteractionData()

		// If user + amount provided → direct XP award/deduction
		if target, ok := data.OptUser("user"); ok {
			amount, amountOk := data.OptInt("amount")
			if !amountOk {
				amount = 250 // default
			}

			userID := int64(target.ID)
			xpDelta := int64(amount)

			// Apply XP delta
			updated, err := queries.AddXPDelta(ctx, db.AddXPDeltaParams{
				GuildId: int64(*e.GuildID()),
				UserId:  strconv.FormatUint(uint64(target.ID), 10),
				Xp:      xpDelta,
			})
			if err != nil {
				return err
			}

			sign := "+"
			if xpDelta < 0 {
				sign = ""
			}

			return e.CreateMessage(discord.NewMessageCreate().WithEmbeds(discord.Embed{
				Title: "Δ Admin Drop",
				Description: fmt.Sprintf(
					"<@%d> received `%s%d XP`\n```\nf(x) = Level %d | Total XP = %d\n```",
					userID, sign, xpDelta, updated.Level, updated.Xp,
				),
				Color: 0x1a1a2e,
				Footer: &discord.EmbedFooter{
					Text: "f(x) — Function • Admin Drop",
				},
			}))
		}

		// No user → trigger a math question drop in current channel
		channelID := uint64(e.Channel().ID())
		adminID := uint64(e.User().ID)
		err := dropSvc.SendDrop(ctx, channelID, &adminID)
		if err != nil {
			return err
		}

		return handler.RespondEphemeral(e, "∫ Drop initiated!")
	}
}
