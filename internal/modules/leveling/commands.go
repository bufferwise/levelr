package leveling

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/handler"
	"github.com/bufferwise/levelr/internal/services"
)

var RankCommand = discord.SlashCommandCreate{
	Name:        "rank",
	Description: "∫ Shows comprehensive user stats with all/ leaderboard positions",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionUser{
			Name:        "user",
			Description: "Target user (optional - defaults to you)",
			Required:    false,
		},
	},
}

func HandleRank(queries *db.Queries, blSvc *services.BlacklistService) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		target := e.User()
		data := e.SlashCommandInteractionData()
		if user, ok := data.OptUser("user"); ok {
			target = user
		}

		userIDStr := strconv.FormatUint(uint64(target.ID), 10)
		guildIDRaw := e.GuildID()
		if guildIDRaw == nil {
			return e.CreateMessage(discord.NewMessageCreate().WithContent("This command can only be used in a server."))
		}
		guildID := int64(*guildIDRaw)

		dbUser, err := queries.GetUser(ctx, db.GetUserParams{
			UserId: userIDStr,
			GuildId: guildID,
		})
		if err != nil {
			// Initialize default for non-existent users
			dbUser = db.UsersXp{
				Xp:         0,
				Level:      0,
				MsgAlltime: 0,
				VcAlltime:  0,
			}
		}

		// Get all ranks and stats
		xpRank, _ := queries.GetServerRank(ctx, db.GetServerRankParams{
			Level: dbUser.Level,
			Xp:    dbUser.Xp,
		})
		// Get weekly stats
		weekStart := services.WeekStartString(time.Now())
		weeklyMsgRank, _ := queries.GetWeeklyMsgRank(ctx, db.GetWeeklyMsgRankParams{
			YearWeek: weekStart,
			GuildId:  strconv.FormatUint(uint64(guildID), 10),
			UserId:   userIDStr,
		})
		weeklyVCRank, _ := queries.GetWeeklyVCRank(ctx, db.GetWeeklyVCRankParams{
			YearWeek: weekStart,
			GuildId:  strconv.FormatUint(uint64(guildID), 10),
			UserId:   userIDStr,
		})

		// Math progress calculation: XP resets to 0, so dbUser.Xp is current progress.
		currentLevel := int(dbUser.Level)
		nextLevel := currentLevel + 1
		xpProgress := dbUser.Xp
		xpForNext := XPForLevel(nextLevel)
		if xpForNext == 0 {
			xpForNext = 100
		}

		percentage := 0.0
		if xpForNext > 0 {
			percentage = float64(xpProgress) / float64(xpForNext) * 100
		}

		// Progress bar (Sleek layout)
		barLen := 10
		filled := int(float64(barLen) * (percentage / 100))
		if filled > barLen {
			filled = barLen
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barLen-filled)

		// Check blacklist status
		blStatus, _ := blSvc.GetFullUserStatus(ctx, strconv.FormatInt(guildID, 10), uint64(target.ID))
		var blBlock string
		if blStatus.IsBlacklisted && !blStatus.IsHidden {
			expiresDesc := "permanently"
			if blStatus.ExpiresAt != nil {
				expiresDesc = fmt.Sprintf("expires <t:%d:R>", blStatus.ExpiresAt.Unix())
			}
			var inheritedDesc string
			if strings.ToLower(blStatus.SourceType) != "user" && blStatus.SourceType != "" {
				inheritedDesc = fmt.Sprintf(" (inherited from %s)", blStatus.SourceType)
			}
			blBlock = fmt.Sprintf(
				"\n\n⚠️ **Note:** You are currently blacklisted from earning XP%s: *%s* (%s).",
				inheritedDesc,
				blStatus.Reason,
				expiresDesc,
			)
		}

		// Build professional sleek minimalistic description
		rankText := fmt.Sprintf(
			"## 📊 **∫ %s's Stats**\n"+
				"**Level %d** • `%d / %d XP` • `%.1f%%`\n"+
				"%s\n\n"+
				"**🏆 Leaderboard Rank:** `#%d`\n"+
				"**📅 Weekly Rank:** `💬 #%d` • `🎙️ #%d`"+
				"%s",
			target.Username,
			currentLevel, xpProgress, xpForNext, percentage,
			bar,
			xpRank,
			weeklyMsgRank,
			weeklyVCRank,
			blBlock,
		)

		return e.CreateMessage(discord.NewMessageCreate().
			WithContent(rankText).
			WithAllowedMentions(&discord.AllowedMentions{
				Parse: []discord.AllowedMentionType{}, // No pings
			}))
	}
}

var LeaderboardCommand = discord.SlashCommandCreate{
	Name:        "leaderboard",
	Description: "∑ Shows the top users in the server",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionString{
			Name:        "type",
			Description: "Leaderboard type",
			Required:    true,
			Choices: []discord.ApplicationCommandOptionChoiceString{
				{Name: "All-time XP (Top 100)", Value: "xp"},
				{Name: "All-time Messages", Value: "msg_all"},
				{Name: "All-time Voice", Value: "vc_all"},
				{Name: "Weekly Messages", Value: "msg_week"},
				{Name: "Weekly Voice", Value: "vc_week"},
			},
		},
	},
}

func buildLeaderboardMessage(ctx context.Context, queries *db.Queries, guildID int64, lbType string, page int) (discord.MessageCreate, []discord.InteractiveComponent) {
	var title string
	var lines []string
	var maxPage int

	limit := 10
	offset := page * limit
	guildIDStr := strconv.FormatUint(uint64(guildID), 10)

	switch lbType {
	case "xp":
		title = "## ∑ Σ(n=1→100) — All-Time Rankings"
		rows, _ := queries.GetGuildLeaderboard(ctx, db.GetGuildLeaderboardParams{
			GuildId: guildID,
			Limit:   100,
		})
		maxPage = (len(rows) - 1) / limit
		if maxPage < 0 {
			maxPage = 0
		}
		if page > maxPage {
			page = maxPage
			offset = page * limit
		}

		end := offset + limit
		if end > len(rows) {
			end = len(rows)
		}

		if offset < len(rows) {
			for i, r := range rows[offset:end] {
				// Display safety: if XP is somehow >= requirement for next level (legacy data),
				// show 0 to maintain the 'reset' aesthetic until activity pushes it.
				displayXP := r.Xp
				nextReq := XPForLevel(int(r.Level + 1))
				if nextReq <= 0 {
					nextReq = 100
				}
				if displayXP >= nextReq {
					displayXP = 0
				}

				line := fmt.Sprintf("%d. <@%s> — Lvl %d (%d XP)", offset+i+1, r.UserIDStr, r.Level, displayXP)
				lines = append(lines, line)
			}
		}
	case "msg_all":
		title = "## 💬 ∂ Messages All-Time"
		maxPage = 9 // Limit to top 100
		rows, _ := queries.GetAllTimeMsgLeaderboard(ctx, db.GetAllTimeMsgLeaderboardParams{Limit: int64(limit), Offset: int64(offset)})
		for i, r := range rows {
			lines = append(lines, fmt.Sprintf("%d. <@%s> — %d messages", offset+i+1, r.UserIDStr, r.MsgAlltime))
		}
	case "vc_all":
		title = "## 🎙️ ∫ Voice All-Time"
		maxPage = 9
		rows, _ := queries.GetAllTimeVCLeaderboard(ctx, db.GetAllTimeVCLeaderboardParams{Limit: int64(limit), Offset: int64(offset)})
		for i, r := range rows {
			lines = append(lines, fmt.Sprintf("%d. <@%s> — %d minutes", offset+i+1, r.UserIDStr, r.VcAlltime))
		}
	case "msg_week":
		title = "## 📅 π Weekly Messages"
		maxPage = 9
		ws := services.WeekStartString(time.Now())
		rows, _ := queries.GetWeeklyMsgLeaderboard(ctx, db.GetWeeklyMsgLeaderboardParams{
			GuildId:  guildIDStr,
			YearWeek: ws,
			Limit:    int64(limit),
			Offset:   int64(offset),
		})
		for i, r := range rows {
			lines = append(lines, fmt.Sprintf("%d. <@%s> — %d messages", offset+i+1, r.Userid, r.Count))
		}
	case "vc_week":
		title = "## 🕒 Δ Weekly Voice"
		maxPage = 9
		ws := services.WeekStartString(time.Now())
		rows, _ := queries.GetWeeklyVCLeaderboard(ctx, db.GetWeeklyVCLeaderboardParams{
			GuildId:  guildIDStr,
			YearWeek: ws,
			Limit:    int64(limit),
			Offset:   int64(offset),
		})
		for i, r := range rows {
			lines = append(lines, fmt.Sprintf("%d. <@%s> — %d minutes", offset+i+1, r.Userid, r.Minutes))
		}
	}

	desc := "∫ No data yet. Be the first to climb the ranks!"
	if len(lines) > 0 {
		desc = strings.Join(lines, "\n")
	}

	header := fmt.Sprintf("**%s**\n----------------------------\n", title)
	content := header + desc + fmt.Sprintf("\n\n*Page %d/%d • Levels reset XP to 0*", page+1, maxPage+1)

	buttons := []discord.InteractiveComponent{
		discord.NewSecondaryButton("◀ Prev", fmt.Sprintf("leaderboard:%d:%s:%d", guildID, lbType, page-1)).WithDisabled(page <= 0),
		discord.NewSecondaryButton("Next ▶", fmt.Sprintf("leaderboard:%d:%s:%d", guildID, lbType, page+1)).WithDisabled(page >= maxPage || len(lines) < limit),
	}

	var messageCreate discord.MessageCreate
	// Always use silent (notifications suppressed) messages for leaderboard
	messageCreate = discord.NewMessageCreate().
		WithContent(content).
		AddActionRow(buttons...).
		WithAllowedMentions(&discord.AllowedMentions{
			Parse: []discord.AllowedMentionType{}, // No pings - always silent
		}).
		WithFlags(discord.MessageFlagSuppressNotifications)

	return messageCreate, buttons
}

func HandleLeaderboard(queries *db.Queries) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		guildID := int64(*e.GuildID())
		data := e.SlashCommandInteractionData()
		lbType := data.String("type")
		msgCreate, buttons := buildLeaderboardMessage(ctx, queries, guildID, lbType, 0)
		err := e.CreateMessage(msgCreate)
		if err == nil {
			// Auto-disable buttons after 10 seconds
			time.AfterFunc(10*time.Second, func() {
				var disabledButtons []discord.InteractiveComponent
				for _, b := range buttons {
					btn := b.(discord.ButtonComponent)
					btn.Disabled = true
					disabledButtons = append(disabledButtons, btn)
				}
				_, _ = e.Client().Rest.UpdateInteractionResponse(e.ApplicationID(), e.Token(), discord.NewMessageUpdate().AddActionRow(disabledButtons...))
			})
		}
		return err
	}
}

func HandleLeaderboardButton(queries *db.Queries) handler.ButtonHandler {
	return func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		parts := strings.Split(e.Data.CustomID(), ":")
		if len(parts) != 4 {
			return nil
		}

		var guildID int64
		fmt.Sscanf(parts[1], "%d", &guildID)
		lbType := parts[2]
		var page int
		fmt.Sscanf(parts[3], "%d", &page)

		msgCreate, buttons := buildLeaderboardMessage(ctx, queries, guildID, lbType, page)

		msgUpdate := discord.NewMessageUpdate().
			WithContent(msgCreate.Content).
			WithEmbeds(msgCreate.Embeds...).
			WithComponents(msgCreate.Components...).
			WithAllowedMentions(msgCreate.AllowedMentions)

		err := e.UpdateMessage(msgUpdate)
		if err == nil {
			// Auto-disable buttons after 10 seconds
			time.AfterFunc(10*time.Second, func() {
				var disabledButtons []discord.InteractiveComponent
				for _, b := range buttons {
					btn := b.(discord.ButtonComponent)
					btn.Disabled = true
					disabledButtons = append(disabledButtons, btn)
				}
				_, _ = e.Client().Rest.UpdateInteractionResponse(e.ApplicationID(), e.Token(), discord.NewMessageUpdate().AddActionRow(disabledButtons...))
			})
		}
		return err
	}
}
