package giveaways

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	db "github.com/bufferwise/levelr/internal/db/sqlc"
)

type WorkerManager struct {
	client     *bot.Client
	queries    *db.Queries
	activeJobs sync.Map // giveawayID (int64) -> *time.Timer
}

func NewWorkerManager(client *bot.Client, queries *db.Queries) *WorkerManager {
	return &WorkerManager{
		client:  client,
		queries: queries,
	}
}

// StartRecoveryWorker scans and schedules all unended giveaways on bot startup.
func (wm *WorkerManager) StartRecoveryWorker(ctx context.Context) {
	slog.Info("Starting giveaways background worker and recovery monitor...")

	active, err := wm.queries.ListActive(ctx)
	if err != nil {
		slog.Error("Failed to list active giveaways on startup", slog.Any("err", err))
		return
	}

	now := time.Now()
	for _, gw := range active {
		giveaway := gw // capture local variable
		durationLeft := giveaway.EndsAt.Sub(now)

		if durationLeft <= 0 {
			slog.Info("Found expired unended giveaway on startup, drawing immediately",
				slog.Int64("giveaway_id", giveaway.ID),
				slog.String("prize", giveaway.Prize),
			)
			go wm.DrawAndEndGiveaway(context.Background(), giveaway.ID)
		} else {
			slog.Info("Scheduling active giveaway draw",
				slog.Int64("giveaway_id", giveaway.ID),
				slog.String("prize", giveaway.Prize),
				slog.Duration("time_left", durationLeft),
			)
			wm.ScheduleGiveaway(giveaway.ID, durationLeft)
		}
	}
}

// ScheduleGiveaway schedules a giveaway drawing routine after the specified duration.
func (wm *WorkerManager) ScheduleGiveaway(giveawayID int64, delay time.Duration) {
	// Cancel any existing job for safety
	wm.CancelJob(giveawayID)

	timer := time.AfterFunc(delay, func() {
		wm.activeJobs.Delete(giveawayID)
		wm.DrawAndEndGiveaway(context.Background(), giveawayID)
	})

	wm.activeJobs.Store(giveawayID, timer)
}

// CancelJob cancels a scheduled drawing job if it exists.
func (wm *WorkerManager) CancelJob(giveawayID int64) {
	if val, ok := wm.activeJobs.Load(giveawayID); ok {
		if timer, ok := val.(*time.Timer); ok {
			timer.Stop()
		}
		wm.activeJobs.Delete(giveawayID)
	}
}

// DrawAndEndGiveaway runs the raffle drawing routine, marks the giveaway as ended in the DB,
// updates the original post, and announces the winners publicly.
func (wm *WorkerManager) DrawAndEndGiveaway(ctx context.Context, giveawayID int64) {
	slog.Info("Executing giveaway drawing routine...", slog.Int64("giveaway_id", giveawayID))

	// Get giveaway details
	giveaway, err := wm.queries.GetGiveaway(ctx, giveawayID)
	if err != nil {
		slog.Error("Failed to fetch giveaway for drawing", slog.Int64("giveaway_id", giveawayID), slog.Any("err", err))
		return
	}

	if giveaway.Ended {
		slog.Warn("Giveaway already marked as ended, skipping duplicate draw", slog.Int64("giveaway_id", giveawayID))
		return
	}

	// Mark ended in database immediately to prevent race conditions or duplicate drawing
	if err := wm.queries.EndGiveaway(ctx, giveawayID); err != nil {
		slog.Error("Failed to set giveaway as ended in database", slog.Int64("giveaway_id", giveawayID), slog.Any("err", err))
		return
	}

	// Get entrants
	entrants, err := wm.queries.ListEntries(ctx, giveawayID)
	if err != nil {
		slog.Error("Failed to fetch entrants for giveaway drawing", slog.Int64("giveaway_id", giveawayID), slog.Any("err", err))
		return
	}

	channelID := snowflake.ID(giveaway.ChannelID)
	messageID := snowflake.ID(giveaway.MessageID)

	// Fetch original message to get the guild ID safely
	var guildID snowflake.ID
	msg, err := wm.client.Rest.GetMessage(channelID, messageID)
	if err == nil && msg.GuildID != nil {
		guildID = *msg.GuildID
	}

	if len(entrants) == 0 {
		slog.Info("No entrants participated in giveaway", slog.Int64("giveaway_id", giveawayID))
		
		// Update original message to show ended status with no entrants
		embed := discord.Embed{
			Title: giveaway.Prize,
			Description: fmt.Sprintf(
				"Winners: **No entrants participated.**\n"+
					"Hosted by: <@%s>\n"+
					"Ended: <t:%d:R>",
				strconv.FormatInt(giveaway.HostID, 10),
				giveaway.EndsAt.Unix(),
			),
			Color: RandomColor(),
		}

		// Update row components to keep Participants active even when ended
		disabledRow := discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    "Giveaway Ended",
				CustomID: "giveaway:ended_btn",
				Disabled: true,
				Emoji:    &discord.ComponentEmoji{Name: "🔒"},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    "Participants",
				CustomID: fmt.Sprintf("giveaway_participants_btn:%d", giveaway.ID),
				Emoji:    &discord.ComponentEmoji{Name: "👥"},
			},
		)

		update := discord.NewMessageUpdate().
			WithEmbeds(embed).
			WithComponents(disabledRow)
		
		_, _ = wm.client.Rest.UpdateMessage(channelID, messageID, update)

		// Send public notification
		_, _ = wm.client.Rest.CreateMessage(channelID, discord.NewMessageCreate().
			WithContent(fmt.Sprintf("😭 No one entered the giveaway for **%s**, so no winner could be drawn!", giveaway.Prize)),
		)
		return
	}

	// Parse role multipliers mapping: {"RoleID": MultiplierFloat}
	multipliers := make(map[snowflake.ID]float64)
	if giveaway.RoleMultipliers != "" {
		var raw map[string]float64
		if err := json.Unmarshal([]byte(giveaway.RoleMultipliers), &raw); err == nil {
			for roleStr, mult := range raw {
				if rID, err := snowflake.Parse(roleStr); err == nil {
					multipliers[rID] = mult
				}
			}
		}
	}

	// Build secure ticket pool using CSPRNG raffle mechanics
	var tickets []snowflake.ID
	for _, entryID := range entrants {
		userSnowflake := snowflake.ID(entryID)
		ticketCount := 10 // Base tickets = 1.0x

		// Check member multipliers if guild is valid
		if guildID != 0 {
			member, err := wm.client.Rest.GetMember(guildID, userSnowflake)
			if err == nil {
				highestMult := 1.0
				for _, rID := range member.RoleIDs {
					if mult, ok := multipliers[rID]; ok {
						if mult > highestMult {
							highestMult = mult
						}
					}
				}
				ticketCount = int(10.0 * highestMult)
			}
		}

		// Fill pool
		for i := 0; i < ticketCount; i++ {
			tickets = append(tickets, userSnowflake)
		}
	}

	// Draw winners cleanly
	winners := DrawWinnersCSPRNG(tickets, int(giveaway.WinnerCount))

	// Format winner mentions
	var winnerMentions []string
	for _, wID := range winners {
		winnerMentions = append(winnerMentions, fmt.Sprintf("<@%s>", wID.String()))
	}

	var winnersStr string
	if len(winnerMentions) > 0 {
		winnersStr = strings.Join(winnerMentions, ", ")
	} else {
		winnersStr = "Failed to determine winners."
	}

	winnerLabel := "Winner"
	if giveaway.WinnerCount > 1 {
		winnerLabel = "Winners"
	}

	embed := discord.Embed{
		Title: giveaway.Prize,
		Description: fmt.Sprintf(
			"%s: %s\n"+
				"Hosted by: <@%s>\n"+
				"Ended at · <t:%d:f>",
			winnerLabel,
			winnersStr,
			strconv.FormatInt(giveaway.HostID, 10),
			giveaway.EndsAt.Unix(),
		),
		Color: RandomColor(),
	}

	// Create action components matching the screenshot
	actionComponents := []discord.InteractiveComponent{
		discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    strconv.Itoa(len(entrants)),
			CustomID: "giveaway:ended_btn",
			Disabled: true,
			Emoji:    &discord.ComponentEmoji{Name: "🎉"},
		},
		discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    "Participants",
			CustomID: fmt.Sprintf("giveaway_participants_btn:%d", giveaway.ID),
			Emoji:    &discord.ComponentEmoji{Name: "👥"},
		},
	}

	update := discord.NewMessageUpdate().
		WithEmbeds(embed).
		WithComponents(discord.NewActionRow(actionComponents...))
	
	_, _ = wm.client.Rest.UpdateMessage(channelID, messageID, update)

	// Send loud public congratulatory notification tagging all winners with the premium congratulations embed
	congratsEmbed := discord.Embed{
		Description: fmt.Sprintf(
			"%s won the giveaway of **%s**!\n\n"+
				"• Hosted by: <@%s>\n"+
				"• Reroll Command: `/giveaway reroll message_id: %d`",
			winnersStr,
			giveaway.Prize,
			strconv.FormatInt(giveaway.HostID, 10),
			giveaway.MessageID,
		),
		Color: RandomColor(),
	}

	// Link Button pointing back to original giveaway post
	msgLink := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildID.String(), channelID.String(), messageID.String())
	congratsRow := discord.NewActionRow(
		discord.ButtonComponent{
			Style: discord.ButtonStyleLink,
			Label: "Giveaway Message",
			URL:   msgLink,
			Emoji: &discord.ComponentEmoji{Name: "↗️"},
		},
		discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    "Reroll",
			CustomID: fmt.Sprintf("giveaway_reroll_btn:%d", giveaway.ID),
			Emoji:    &discord.ComponentEmoji{Name: "🔄"},
		},
	)

	_, _ = wm.client.Rest.CreateMessage(channelID, discord.NewMessageCreate().
		WithContent("Congratulations! 🎉").
		WithEmbeds(congratsEmbed).
		WithComponents(congratsRow),
	)
}

// PerformReroll wraps DrawReroll for button interactions.
func (wm *WorkerManager) PerformReroll(ctx context.Context, e *events.ComponentInteractionCreate, giveawayID int64) error {
	giveaway, err := wm.queries.GetGiveaway(ctx, giveawayID)
	if err != nil {
		return fmt.Errorf("failed to fetch giveaway: %w", err)
	}

	isHost := snowflake.ID(giveaway.HostID) == e.User().ID
	isAdmin := false
	if e.Member() != nil {
		isAdmin = e.Member().Permissions.Has(discord.PermissionManageGuild) || e.Member().Permissions.Has(discord.PermissionAdministrator)
	}

	winners, err := wm.DrawReroll(ctx, *e.GuildID(), e.User().ID, isHost, isAdmin, giveawayID)
	if err != nil {
		return e.CreateMessage(discord.NewMessageCreate().
			WithContent(fmt.Sprintf("✗ %v", err)).
			WithFlags(discord.MessageFlagEphemeral),
		)
	}

	return e.CreateMessage(discord.NewMessageCreate().
		WithContent(fmt.Sprintf("✓ Successfully rerolled! Winner(s): %s", winners)).
		WithFlags(discord.MessageFlagEphemeral),
	)
}

// DrawReroll draws new winners from the existing ticket pool of an ended giveaway.
func (wm *WorkerManager) DrawReroll(ctx context.Context, guildID snowflake.ID, triggerUserID snowflake.ID, isHost bool, isAdmin bool, giveawayID int64) (string, error) {
	giveaway, err := wm.queries.GetGiveaway(ctx, giveawayID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch giveaway: %w", err)
	}

	if !isHost && !isAdmin {
		return "", fmt.Errorf("only the host or server administrators can reroll winners")
	}

	entrants, err := wm.queries.ListEntries(ctx, giveawayID)
	if err != nil || len(entrants) == 0 {
		return "", fmt.Errorf("no active entries found")
	}

	chID := snowflake.ID(giveaway.ChannelID)
	mID := snowflake.ID(giveaway.MessageID)

	// Fetch current winners from the original message's embed description to exclude them
	currentWinners := make(map[snowflake.ID]bool)
	msg, err := wm.client.Rest.GetMessage(chID, mID)
	if err == nil && len(msg.Embeds) > 0 {
		desc := msg.Embeds[0].Description
		lines := strings.Split(desc, "\n")
		if len(lines) > 0 {
			firstLine := lines[0]
			temp := firstLine
			for {
				idx := strings.Index(temp, "<@")
				if idx == -1 {
					break
				}
				temp = temp[idx+2:]
				endIdx := strings.Index(temp, ">")
				if endIdx == -1 {
					break
				}
				userIDStr := temp[:endIdx]
				userIDStr = strings.TrimLeft(userIDStr, "!&")
				if uID, err := snowflake.Parse(userIDStr); err == nil {
					currentWinners[uID] = true
				}
				temp = temp[endIdx+1:]
			}
		}
	}

	// Parse role multipliers mapping
	multipliers := make(map[snowflake.ID]float64)
	if giveaway.RoleMultipliers != "" {
		var raw map[string]float64
		if err := json.Unmarshal([]byte(giveaway.RoleMultipliers), &raw); err == nil {
			for roleStr, mult := range raw {
				if rID, err := snowflake.Parse(roleStr); err == nil {
					multipliers[rID] = mult
				}
			}
		}
	}

	// Re-build ticket pool
	var tickets []snowflake.ID
	for _, entryID := range entrants {
		userSnowflake := snowflake.ID(entryID)
		if currentWinners[userSnowflake] {
			continue
		}
		ticketCount := 10 // Base tickets = 1.0x

		member, err := wm.client.Rest.GetMember(guildID, userSnowflake)
		if err == nil {
			highestMult := 1.0
			for _, rID := range member.RoleIDs {
				if mult, ok := multipliers[rID]; ok {
					if mult > highestMult {
						highestMult = mult
					}
				}
			}
			ticketCount = int(10.0 * highestMult)
		}

		for i := 0; i < ticketCount; i++ {
			tickets = append(tickets, userSnowflake)
		}
	}

	if len(tickets) == 0 {
		return "", fmt.Errorf("all participants have already won or no other eligible entrants are available")
	}

	// Draw new winners
	winners := DrawWinnersCSPRNG(tickets, int(giveaway.WinnerCount))

	var winnerMentions []string
	for _, wID := range winners {
		winnerMentions = append(winnerMentions, fmt.Sprintf("<@%s>", wID.String()))
	}

	var winnersStr string
	if len(winnerMentions) > 0 {
		winnersStr = strings.Join(winnerMentions, ", ")
	} else {
		winnersStr = "No valid winners could be drawn."
	}

	winnerLabel := "Winner (Rerolled)"
	if giveaway.WinnerCount > 1 {
		winnerLabel = "Winners (Rerolled)"
	}

	embed := discord.Embed{
		Title: giveaway.Prize,
		Description: fmt.Sprintf(
			"%s: %s\n"+
				"Hosted by: <@%s>\n"+
				"Ended at · <t:%d:f>",
			winnerLabel,
			winnersStr,
			strconv.FormatInt(giveaway.HostID, 10),
			giveaway.EndsAt.Unix(),
		),
		Color: RandomColor(),
	}

	actionComponents := []discord.InteractiveComponent{
		discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    strconv.Itoa(len(entrants)),
			CustomID: "giveaway:ended_btn",
			Disabled: true,
			Emoji:    &discord.ComponentEmoji{Name: "🎉"},
		},
		discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    "Participants",
			CustomID: fmt.Sprintf("giveaway_participants_btn:%d", giveaway.ID),
			Emoji:    &discord.ComponentEmoji{Name: "👥"},
		},
	}

	// Update original post
	_, _ = wm.client.Rest.UpdateMessage(chID, mID, discord.NewMessageUpdate().
		WithEmbeds(embed).
		WithComponents(discord.NewActionRow(actionComponents...)),
	)

	// Send public congratulations message for reroll winners with link button and reroll button
	congratsEmbed := discord.Embed{
		Description: fmt.Sprintf(
			"%s won the giveaway of **%s**!\n\n"+
				"• Hosted by: <@%s>\n"+
				"• Reroll Command: `/giveaway reroll message_id: %d`",
			winnersStr,
			giveaway.Prize,
			strconv.FormatInt(giveaway.HostID, 10),
			giveaway.MessageID,
		),
		Color: RandomColor(),
	}

	msgLink := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildID.String(), chID.String(), mID.String())
	congratsRow := discord.NewActionRow(
		discord.ButtonComponent{
			Style: discord.ButtonStyleLink,
			Label: "Giveaway Message",
			URL:   msgLink,
			Emoji: &discord.ComponentEmoji{Name: "↗️"},
		},
		discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    "Reroll",
			CustomID: fmt.Sprintf("giveaway_reroll_btn:%d", giveaway.ID),
			Emoji:    &discord.ComponentEmoji{Name: "🔄"},
		},
	)

	_, _ = wm.client.Rest.CreateMessage(chID, discord.NewMessageCreate().
		WithContent("Congratulations! 🎉").
		WithEmbeds(congratsEmbed).
		WithComponents(congratsRow),
	)

	return winnersStr, nil
}
