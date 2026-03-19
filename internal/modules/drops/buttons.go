package drops

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/bufferwise/levelr/internal/handler"
)

// HandleDropButton handles button interactions for math drop MCQs.
func HandleDropButton(svc *DropService) handler.ButtonHandler {
	return func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		customID := e.Data.CustomID()

		userID := uint64(e.User().ID)
		messageID := uint64(e.Message.ID)
		guildID := uint64(0)
		if e.GuildID() != nil {
			guildID = uint64(*e.GuildID())
		}

		winnerID, correct, xpAmount, err := svc.ClaimDrop(ctx, messageID, userID, guildID, customID)
		if err != nil {
			return handler.RespondEphemeral(e, "∂ Error processing your answer.")
		}

		if winnerID != 0 && winnerID != userID {
			// Correct but already claimed by someone else (or already claimed by someone else in general)
			return handler.RespondEphemeral(e, fmt.Sprintf("∫ Too late! This drop was already claimed by <@%d>.", winnerID))
		}

		if !correct {
			// Wrong answer — ephemeral feedback
			return handler.RespondEphemeral(e, "✗ Incorrect. Try again!")
		}

		// Winner! Disable all buttons and announce
		var disabledButtons []discord.InteractiveComponent
		labels := []string{"A", "B", "C", "D"}
		for i := 0; i < 4; i++ {
			style := discord.ButtonStyleSecondary
			disabledButtons = append(disabledButtons, discord.ButtonComponent{
				Style:    style,
				Label:    labels[i],
				CustomID: fmt.Sprintf("drop_%s", labels[i]),
				Disabled: true,
			})
		}

		// Update the original message to disable buttons
		updateBuilder := discord.NewMessageUpdate().
			AddActionRow(disabledButtons...)

		_, _ = e.Client().Rest.UpdateMessage(e.Message.ChannelID, e.Message.ID, updateBuilder)

		// Send winner announcement with math flair
		embed := discord.Embed{
			Title: "∑ Drop Claimed!",
			Description: fmt.Sprintf(
				"<@%d> solved the equation and earned **%d XP**!\n\n`f(x) = x + %d`",
				userID, xpAmount, xpAmount,
			),
			Color: 0x2ecc71, // green for success
			Footer: &discord.EmbedFooter{
				Text: "f(x) — Function",
			},
		}

		return e.CreateMessage(discord.NewMessageCreate().WithEmbeds(embed))
	}
}
