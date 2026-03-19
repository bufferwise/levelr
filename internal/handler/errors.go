package handler

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
)

// InteractionResponder is the common interface for responding to interaction events.
type InteractionResponder interface {
	CreateMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) error
}

// RespondError sends an ephemeral error message to the user.
// It logs the original error with slog and masks the real error string from the user,
// presenting the safe, user-friendly prompt string.
func RespondError(e InteractionResponder, msg string, err error) {
	if err != nil {
		slog.Error("interaction handler error", slog.Any("err", err))
	}
	_ = e.CreateMessage(discord.NewMessageCreate().
		WithContent("✗ " + msg).
		WithFlags(discord.MessageFlagEphemeral))
}

// RespondEphemeral sends a simple ephemeral text reply.
func RespondEphemeral(e InteractionResponder, msg string) error {
	return e.CreateMessage(discord.NewMessageCreate().
		WithContent(msg).
		WithFlags(discord.MessageFlagEphemeral))
}
