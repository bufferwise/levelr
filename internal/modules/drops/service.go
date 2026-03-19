package drops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"

	"github.com/bufferwise/levelr/internal/config"
	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/modules/leveling"
)

// DropService handles math-question XP drops.
type DropService struct {
	queries *db.Queries
	client  *bot.Client
	cfg     *config.Config
}

// NewDropService creates a new DropService.
func NewDropService(queries *db.Queries, client *bot.Client, cfg *config.Config) *DropService {
	return &DropService{cfg: cfg, queries: queries, client: client}
}

// SendDrop generates a math question and sends it as a drop in the given channel.
func (s *DropService) SendDrop(ctx context.Context, channelID uint64, droppedBy *uint64) error {
	q := Random()
	xpAmount := int64(s.cfg.DropXPMin + rand.IntN(s.cfg.DropXPMax-s.cfg.DropXPMin+1))

	// Build LaTeX image URL for the question
	imgURL := LaTeXImageURL(q.LaTeX)

	// Build the embed with mathematics aesthetic
	embed := discord.Embed{
		Title: "Δ XP Drop — f(x) = x + Δ",
		Description: fmt.Sprintf(
			"```\n∫ Solve the following to claim %d XP\n```\n**%s**",
			xpAmount, q.Text,
		),
		Color: 0x1a1a2e, // deep navy
		Image: &discord.EmbedResource{
			URL: imgURL,
		},
		Footer: &discord.EmbedFooter{
			Text: "f(x) — Function • First correct answer wins",
		},
	}

	// Build A/B/C/D buttons
	var buttons []discord.InteractiveComponent
	labels := []string{"A", "B", "C", "D"}
	styles := []discord.ButtonStyle{
		discord.ButtonStylePrimary,
		discord.ButtonStylePrimary,
		discord.ButtonStylePrimary,
		discord.ButtonStylePrimary,
	}
	for i := 0; i < 4; i++ {
		buttons = append(buttons, discord.ButtonComponent{
			Style:    styles[i],
			Label:    fmt.Sprintf("%s) %s", labels[i], q.Options[i]),
			CustomID: fmt.Sprintf("drop_%s", labels[i]),
		})
	}

	msg, err := (*s.client).Rest.CreateMessage(
		snowflake.ID(channelID),
		discord.NewMessageCreate().
			WithEmbeds(embed).
			AddActionRow(buttons...),
	)
	if err != nil {
		return fmt.Errorf("send drop message: %w", err)
	}

	// Auto-disable buttons after 10 seconds
	time.AfterFunc(10*time.Second, func() {
		var disabledButtons []discord.InteractiveComponent
		for _, b := range buttons {
			btn := b.(discord.ButtonComponent)
			btn.Disabled = true
			disabledButtons = append(disabledButtons, btn)
		}
		_, _ = (*s.client).Rest.UpdateMessage(snowflake.ID(channelID), msg.ID, discord.NewMessageUpdate().AddActionRow(disabledButtons...))
	})

	// Store the drop in DB
	var droppedBySql sql.NullInt64
	if droppedBy != nil {
		droppedBySql = sql.NullInt64{Int64: int64(*droppedBy), Valid: true}
	}

	_, err = s.queries.InsertDrop(ctx, db.InsertDropParams{
		Question:  q.Text,
		Answer:    AnswerLabel(q.Answer),
		XpAmount:  xpAmount,
		DroppedBy: droppedBySql,
		ChannelID: int64(channelID),
		MessageID: sql.NullInt64{Int64: int64(msg.ID), Valid: true},
	})
	if err != nil {
		slog.Error("failed to insert drop record", slog.Any("err", err))
	}

	slog.Info("drop sent",
		slog.Uint64("channel_id", channelID),
		slog.Int64("xp", xpAmount),
		slog.String("answer", AnswerLabel(q.Answer)),
	)
	return nil
}

// ClaimDrop handles a button press — checks answer, awards XP, disables buttons.
func (s *DropService) ClaimDrop(ctx context.Context, messageID uint64, userID uint64, guildID uint64, buttonID string) (winnerID uint64, correct bool, xp int64, err error) {
	// Get the drop record
	drop, err := s.queries.GetPendingDropByMessage(ctx, sql.NullInt64{Int64: int64(messageID), Valid: true})
	if err != nil {
		return 0, false, 0, fmt.Errorf("drop not found: %w", err)
	}

	// Extract answer letter from button ID (e.g., "drop_A" → "A")
	pressed := ""
	if len(buttonID) > 5 {
		pressed = buttonID[5:]
	}

	correct = (pressed == drop.Answer)

	// Already claimed?
	if drop.WinnerID.Valid {
		return uint64(drop.WinnerID.Int64), correct, drop.XpAmount, nil
	}

	// Check if correct
	if !correct {
		return 0, false, 0, nil
	}

	claimed_drop, err := s.queries.ClaimDrop(ctx, db.ClaimDropParams{
		ID:       drop.ID,
		WinnerID: sql.NullInt64{Int64: int64(userID), Valid: true},
	})
	if err != nil {
		if fmt.Sprint(err) == "no rows in result set" || err == sql.ErrNoRows {
			dropAfter, _ := s.queries.GetPendingDropByMessage(ctx, sql.NullInt64{Int64: int64(messageID), Valid: true})
			if dropAfter.WinnerID.Valid {
				return uint64(dropAfter.WinnerID.Int64), true, drop.XpAmount, nil
			}
			return 0, true, 0, nil
		}
		return 0, true, 0, fmt.Errorf("claim drop: %w", err)
	}
	if !claimed_drop.WinnerID.Valid {
		dropAfter, _ := s.queries.GetPendingDropByMessage(ctx, sql.NullInt64{Int64: int64(messageID), Valid: true})
		if dropAfter.WinnerID.Valid {
			return uint64(dropAfter.WinnerID.Int64), true, drop.XpAmount, nil
		}
		return 0, true, 0, nil // race condition, someone else claimed
	}

	// Award XP
	user, _ := s.queries.GetUser(ctx, db.GetUserParams{
		UserId:  strconv.FormatUint(userID, 10),
		GuildId: int64(guildID),
	})
	oldLevel := int(user.Level)

	updatedUser, err := s.queries.AddXPDelta(ctx, db.AddXPDeltaParams{
		GuildId: int64(guildID),
		UserId:  strconv.FormatUint(userID, 10),
		Xp:     drop.XpAmount,
	})
	if err != nil {
		slog.Error("failed to award drop XP", slog.Any("err", err))
	}

	// Recompute level
	newLevel := leveling.CurrentLevel(updatedUser.Xp)
	if newLevel != int(updatedUser.Level) { // Level field
		_ = s.queries.UpdateUserLevel(ctx, db.UpdateUserLevelParams{
			Xp:      int64(newLevel), // Using Xp field to store level in old schema
			UserId:  strconv.FormatUint(userID, 10),
			GuildId: int64(guildID),
		})
	}

	// Check level up
	if newLevel > oldLevel {
		leveling.CheckLevelUp(ctx, s.client, snowflake.ID(guildID), snowflake.ID(userID), oldLevel, newLevel, func(_ context.Context, _, _ snowflake.ID, _ int) {})
	}

	return uint64(userID), true, drop.XpAmount, nil
}
