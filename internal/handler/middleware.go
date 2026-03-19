package handler

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/bufferwise/levelr/internal/config"
)

// Middleware wraps a CommandHandler with pre/post processing capabilities.
type Middleware func(next CommandHandler) CommandHandler

// GuildGuard returns a middleware that guarantees the event was sent
// within the context of the centrally configured main guild.
func GuildGuard(cfg *config.Config) Middleware {
	return func(next CommandHandler) CommandHandler {
		return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
			if e.GuildID() == nil || uint64(*e.GuildID()) != cfg.MainGuildID {
				return nil // Silently ignore interactions from unexpected / out-of-guild contexts
			}
			return next(ctx, e)
		}
	}
}

// ErrorCatcher evaluates incoming panic recoveries and command evaluation errors,
// standardizing responses internally, maintaining uptime context boundary logic.
func ErrorCatcher() Middleware {
	return func(next CommandHandler) CommandHandler {
		return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic in handler: %v", r)
					slog.Error("panic recovered", slog.Any("panic", r), slog.String("stack", string(debug.Stack())))
					RespondError(e, "An unexpected internal error occurred.", err)
				}
			}()

			err = next(ctx, e)
			if err != nil {
				RespondError(e, "An error occurred while executing this command.", err)
			}
			return err
		}
	}
}

// PermissionGuard returns a middleware evaluating explicit discord permissions logic context before continuing.
func PermissionGuard(perm discord.Permissions) Middleware {
	return func(next CommandHandler) CommandHandler {
		return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
			if e.Member() == nil || !e.Member().Permissions.Has(perm) {
				RespondError(e, "You do not have permission to use this command.", nil)
				return nil
			}
			return next(ctx, e)
		}
	}
}
