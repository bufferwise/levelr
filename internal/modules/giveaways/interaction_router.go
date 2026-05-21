package giveaways

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/handler"
)

func Ptr[T any](v T) *T {
	return &v
}

func intPtr(v int) *int {
	return &v
}

// buildControlPanelEmbed builds a gorgeous, highly visual card displaying current draft state.
func buildControlPanelEmbed(s *DraftSession) discord.Embed {
	roleReq := "None"
	if s.RequiredRole != nil {
		roleReq = fmt.Sprintf("<@&%s>", s.RequiredRole.String())
	}

	var mults []string
	for rID, mult := range s.RoleMultipliers {
		mults = append(mults, fmt.Sprintf("<@&%s>: **%.1fx**", rID.String(), mult))
	}
	multsStr := "None"
	if len(mults) > 0 {
		multsStr = strings.Join(mults, ", ")
	}

	return discord.Embed{
		Title:       "🛠️ **Giveaway Configuration Wizard**",
		Description: "> Configure your giveaway details and gates below before launching.",
		Color:       0x9b59b6, // Sleek deep purple
		Fields: []discord.EmbedField{
			{Name: "🏆 Prize", Value: fmt.Sprintf("`%s`", s.Prize), Inline: Ptr(true)},
			{Name: "👥 Winner Count", Value: fmt.Sprintf("`%d`", s.WinnerCount), Inline: Ptr(true)},
			{Name: "⏰ Duration", Value: fmt.Sprintf("`%s`", s.Duration.String()), Inline: Ptr(true)},
			{Name: "🔒 Level Requirement", Value: fmt.Sprintf("Level `%d`", s.RequiredLevel), Inline: Ptr(true)},
			{Name: "📅 Min Account Age", Value: fmt.Sprintf("`%d` days", s.MinAccountDays), Inline: Ptr(true)},
			{Name: "🔑 Required Role", Value: roleReq, Inline: Ptr(true)},
			{Name: "⚖️ Booster Role Multipliers", Value: multsStr, Inline: Ptr(false)},
		},
		Footer: &discord.EmbedFooter{
			Text: "Click options below to customize Gating & Boosts",
		},
	}
}

// buildControlPanelComponents constructs action rows containing the native select menu and action buttons.
func buildControlPanelComponents() []discord.LayoutComponent {
	roleSelect := discord.RoleSelectMenuComponent{
		CustomID:    "giveaway_role_select",
		Placeholder: "Select Required Role Gating (Optional)",
		MinValues:   intPtr(0),
		MaxValues:   1,
	}

	btnRow := discord.NewActionRow(
		discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    "Set Level Req",
			CustomID: "giveaway_set_level_btn",
			Emoji:    &discord.ComponentEmoji{Name: "📊"},
		},
		discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    "Set Min Age",
			CustomID: "giveaway_set_min_age_btn",
			Emoji:    &discord.ComponentEmoji{Name: "📅"},
		},
		discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    "Add Multiplier",
			CustomID: "giveaway_set_mults_btn",
			Emoji:    &discord.ComponentEmoji{Name: "⚖️"},
		},
	)

	controlRow := discord.NewActionRow(
		discord.ButtonComponent{
			Style:    discord.ButtonStyleSuccess,
			Label:    "Launch Giveaway 🚀",
			CustomID: "giveaway_launch_btn",
		},
		discord.ButtonComponent{
			Style:    discord.ButtonStyleDanger,
			Label:    "Cancel Setup ❌",
			CustomID: "giveaway_cancel_btn",
		},
	)

	return []discord.LayoutComponent{
		discord.NewActionRow(roleSelect),
		btnRow,
		controlRow,
	}
}

// HandleGiveawayCreateSlash initiates the giveaway interactive setup flow.
func HandleGiveawayCreateSlash(sm *SessionManager) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		guildID := e.GuildID()
		if guildID == nil {
			return handler.RespondEphemeral(e, "✗ This command can only be used inside a server.")
		}

		// Authorization Check: Manage Guild or Administrator
		if e.Member() == nil || (!e.Member().Permissions.Has(discord.PermissionManageGuild) && !e.Member().Permissions.Has(discord.PermissionAdministrator)) {
			return handler.RespondEphemeral(e, "✗ You do not have permission to manage giveaways.")
		}

		// Launch the initial parameters modal
		modal := discord.ModalCreate{
			CustomID: "giveaway_create_modal",
			Title:    "Create Giveaway 🎁",
			Components: []discord.LayoutComponent{
				discord.NewLabel("Prize Description", discord.TextInputComponent{
					CustomID:    "prize",
					Style:       discord.TextInputStyleShort,
					Placeholder: "e.g., $50 Steam Giftcard",
					Required:    true,
				}),
				discord.NewLabel("Duration (e.g. 10m, 1h, 1d)", discord.TextInputComponent{
					CustomID:    "duration",
					Style:       discord.TextInputStyleShort,
					Placeholder: "e.g., 2h",
					Required:    true,
				}),
				discord.NewLabel("Winner Count (Defaults to 1)", discord.TextInputComponent{
					CustomID:    "winner_count",
					Style:       discord.TextInputStyleShort,
					Placeholder: "e.g., 1",
					Required:    false,
				}),
			},
		}

		return e.Modal(modal)
	}
}

// HandleCreateModal handles the submission of the initial parameters modal.
func HandleCreateModal(sm *SessionManager) handler.ModalHandler {
	return func(ctx context.Context, e *events.ModalSubmitInteractionCreate) error {
		prize := e.Data.Text("prize")
		durationStr := e.Data.Text("duration")
		winnersStr := e.Data.Text("winner_count")

		duration, err := parseDuration(durationStr)
		if err != nil {
			return handler.RespondEphemeral(e, "✗ Invalid duration format! Use formats like '10m', '2h', or '1d'.")
		}

		winners := 1
		if winnersStr != "" {
			if parsed, err := strconv.Atoi(winnersStr); err == nil && parsed > 0 {
				winners = parsed
			}
		}

		// Initialize session draft
		session := &DraftSession{
			GuildID:         *e.GuildID(),
			ChannelID:       e.Channel().ID(),
			HostID:          e.User().ID,
			Prize:           prize,
			Duration:        duration,
			WinnerCount:     winners,
			RoleMultipliers: make(map[snowflake.ID]float64),
			State:           "main",
		}

		sm.Set(*e.GuildID(), e.User().ID, session)

		embed := buildControlPanelEmbed(session)
		components := buildControlPanelComponents()

		return e.CreateMessage(discord.NewMessageCreate().
			WithEmbeds(embed).
			WithComponents(components...).
			WithEphemeral(true),
		)
	}
}

// HandleRoleSelect processes role selection updates for required role gating.
func HandleRoleSelect(sm *SessionManager) handler.SelectMenuHandler {
	return func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		session, ok := sm.Get(*e.GuildID(), e.User().ID)
		if !ok {
			return handler.RespondEphemeral(e, "✗ No active giveaway setup session found.")
		}

		data := e.RoleSelectMenuInteractionData()
		if len(data.Values) > 0 {
			rID := data.Values[0]
			session.RequiredRole = &rID
		} else {
			session.RequiredRole = nil
		}

		sm.Set(*e.GuildID(), e.User().ID, session)

		embed := buildControlPanelEmbed(session)
		return e.UpdateMessage(discord.NewMessageUpdate().WithEmbeds(embed))
	}
}

// HandleLevelBtn opens the Level Gating modal.
func HandleLevelBtn(sm *SessionManager) handler.ButtonHandler {
	return func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		session, ok := sm.Get(*e.GuildID(), e.User().ID)
		if !ok {
			return handler.RespondEphemeral(e, "✗ No active giveaway setup session found.")
		}

		modal := discord.ModalCreate{
			CustomID: "giveaway_level_req_modal",
			Title:    "Level Gating 📊",
			Components: []discord.LayoutComponent{
				discord.NewLabel("Minimum Level Requirement", discord.TextInputComponent{
					CustomID:    "min_level",
					Style:       discord.TextInputStyleShort,
					Placeholder: "e.g., 5 (0 to disable)",
					Required:    true,
					Value:       strconv.Itoa(session.RequiredLevel),
				}),
			},
		}

		return e.Modal(modal)
	}
}

// HandleLevelModal processes level requirements configuration submissions.
func HandleLevelModal(sm *SessionManager) handler.ModalHandler {
	return func(ctx context.Context, e *events.ModalSubmitInteractionCreate) error {
		session, ok := sm.Get(*e.GuildID(), e.User().ID)
		if !ok {
			return handler.RespondEphemeral(e, "✗ No active giveaway setup session found.")
		}

		levelStr := e.Data.Text("min_level")
		level := 0
		if levelStr != "" {
			if parsed, err := strconv.Atoi(levelStr); err == nil && parsed >= 0 {
				level = parsed
			}
		}

		session.RequiredLevel = level
		sm.Set(*e.GuildID(), e.User().ID, session)

		embed := buildControlPanelEmbed(session)
		return e.UpdateMessage(discord.NewMessageUpdate().WithEmbeds(embed))
	}
}

// HandleMinAgeBtn opens the Account Age Gating modal.
func HandleMinAgeBtn(sm *SessionManager) handler.ButtonHandler {
	return func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		session, ok := sm.Get(*e.GuildID(), e.User().ID)
		if !ok {
			return handler.RespondEphemeral(e, "✗ No active giveaway setup session found.")
		}

		modal := discord.ModalCreate{
			CustomID: "giveaway_min_age_modal",
			Title:    "Account Age Gating 📅",
			Components: []discord.LayoutComponent{
				discord.NewLabel("Min Account Age (In Days)", discord.TextInputComponent{
					CustomID:    "min_age",
					Style:       discord.TextInputStyleShort,
					Placeholder: "e.g., 7 (0 to disable)",
					Required:    true,
					Value:       strconv.Itoa(session.MinAccountDays),
				}),
			},
		}

		return e.Modal(modal)
	}
}

// HandleMinAgeModal processes account age gating submissions.
func HandleMinAgeModal(sm *SessionManager) handler.ModalHandler {
	return func(ctx context.Context, e *events.ModalSubmitInteractionCreate) error {
		session, ok := sm.Get(*e.GuildID(), e.User().ID)
		if !ok {
			return handler.RespondEphemeral(e, "✗ No active giveaway setup session found.")
		}

		ageStr := e.Data.Text("min_age")
		age := 0
		if ageStr != "" {
			if parsed, err := strconv.Atoi(ageStr); err == nil && parsed >= 0 {
				age = parsed
			}
		}

		session.MinAccountDays = age
		sm.Set(*e.GuildID(), e.User().ID, session)

		embed := buildControlPanelEmbed(session)
		return e.UpdateMessage(discord.NewMessageUpdate().WithEmbeds(embed))
	}
}

// HandleMultsBtn opens the role multiplier configurator modal.
func HandleMultsBtn(sm *SessionManager) handler.ButtonHandler {
	return func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		_, ok := sm.Get(*e.GuildID(), e.User().ID)
		if !ok {
			return handler.RespondEphemeral(e, "✗ No active giveaway setup session found.")
		}

		modal := discord.ModalCreate{
			CustomID: "giveaway_multiplier_modal",
			Title:    "Add Role Multiplier ⚖️",
			Components: []discord.LayoutComponent{
				discord.NewLabel("Role ID to boost", discord.TextInputComponent{
					CustomID:    "role_id",
					Style:       discord.TextInputStyleShort,
					Placeholder: "Paste the Role ID here",
					Required:    true,
				}),
				discord.NewLabel("Luck Multiplier (e.g. 1.5, 2.0)", discord.TextInputComponent{
					CustomID:    "multiplier",
					Style:       discord.TextInputStyleShort,
					Placeholder: "e.g., 2.0",
					Required:    true,
				}),
			},
		}

		return e.Modal(modal)
	}
}

// HandleMultsModal processes multiplier role additions.
func HandleMultsModal(sm *SessionManager) handler.ModalHandler {
	return func(ctx context.Context, e *events.ModalSubmitInteractionCreate) error {
		session, ok := sm.Get(*e.GuildID(), e.User().ID)
		if !ok {
			return handler.RespondEphemeral(e, "✗ No active giveaway setup session found.")
		}

		roleIDStr := e.Data.Text("role_id")
		multStr := e.Data.Text("multiplier")

		roleID, err := snowflake.Parse(roleIDStr)
		if err != nil {
			return handler.RespondEphemeral(e, "✗ Invalid Role ID format!")
		}

		mult, err := strconv.ParseFloat(multStr, 64)
		if err != nil || mult < 1.0 {
			return handler.RespondEphemeral(e, "✗ Invalid multiplier! Must be a decimal number >= 1.0.")
		}

		session.RoleMultipliers[roleID] = mult
		sm.Set(*e.GuildID(), e.User().ID, session)

		embed := buildControlPanelEmbed(session)
		return e.UpdateMessage(discord.NewMessageUpdate().WithEmbeds(embed))
	}
}

// HandleCancelBtn cancels and purges the setup session.
func HandleCancelBtn(sm *SessionManager) handler.ButtonHandler {
	return func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		sm.Delete(*e.GuildID(), e.User().ID)
		return e.UpdateMessage(discord.NewMessageUpdate().
			WithContent("✗ Giveaway setup cancelled.").
			ClearEmbeds().
			ClearComponents(),
		)
	}
}

// HandleLaunchBtn posts the official public giveaway post and schedules the drawing event.
func HandleLaunchBtn(queries *db.Queries, sm *SessionManager, wm *WorkerManager) handler.ButtonHandler {
	return func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		session, ok := sm.Get(*e.GuildID(), e.User().ID)
		if !ok {
			return handler.RespondEphemeral(e, "✗ No active giveaway setup session found.")
		}

		// Calculate timestamps
		endsAt := time.Now().Add(session.Duration)

		// Create the public giveaway embed matching the layout precisely
		embed := discord.Embed{
			Title: session.Prize,
			Description: fmt.Sprintf(
				"Click 🎉 button to enter!\n"+
					"Winners: **%d**\n"+
					"Hosted by: <@%s>\n"+
					"Ends: <t:%d:R> (Timer)\n\n",
				session.WinnerCount,
				session.HostID.String(),
				endsAt.Unix(),
			),
			Color: RandomColor(),
		}

		// Gating & Requirements block (only if configured)
		var gatingLines []string
		if session.RequiredRole != nil {
			gatingLines = append(gatingLines, fmt.Sprintf("Must have the role: <@&%s>", session.RequiredRole.String()))
		}
		if session.RequiredLevel > 0 {
			gatingLines = append(gatingLines, fmt.Sprintf("Must have the minimum level: **Level %d+**", session.RequiredLevel))
		}
		if session.MinAccountDays > 0 {
			gatingLines = append(gatingLines, fmt.Sprintf("Must have the account age: **%d+ days**", session.MinAccountDays))
		}

		if len(gatingLines) > 0 {
			embed.Description += strings.Join(gatingLines, "\n") + "\n\n"
		}
		embed.Description += fmt.Sprintf("Ends at · %s", endsAt.Format("02-01-2006"))

		// Boosts Block (inside fields to keep description clean and identical to target screenshot)
		var boosts []string
		for rID, mult := range session.RoleMultipliers {
			boosts = append(boosts, fmt.Sprintf("<@&%s>: **%.1fx chance**", rID.String(), mult))
		}
		if len(boosts) > 0 {
			embed.Fields = append(embed.Fields, discord.EmbedField{
				Name:  "**Booster Multipliers:**",
				Value: strings.Join(boosts, "\n"),
			})
		}

		// Serialize role multipliers to JSON
		multsJSON, _ := json.Marshal(session.RoleMultipliers)

		// Insert into Database with a temporary Message ID of 0 first to obtain the DB auto-increment ID.
		// This guarantees that the public message's buttons are created with the correct giveaway ID,
		// completely eliminating the need for an additional UpdateMessage API request, preventing rate limits,
		// race conditions, or "pending" custom ID parsing bugs.
		var requiredRoleNull sql.NullInt64
		if session.RequiredRole != nil {
			requiredRoleNull = sql.NullInt64{Int64: int64(*session.RequiredRole), Valid: true}
		}

		gw, err := queries.InsertGiveaway(ctx, db.InsertGiveawayParams{
			ChannelID:       int64(session.ChannelID),
			MessageID:       0, // temporary placeholder
			Prize:           session.Prize,
			WinnerCount:     int64(session.WinnerCount),
			RequiredRole:    requiredRoleNull,
			HostID:          int64(session.HostID),
			EndsAt:          endsAt,
			MinLevel:        int64(session.RequiredLevel),
			MinAccountDays:  int64(session.MinAccountDays),
			RoleMultipliers: string(multsJSON),
		})
		if err != nil {
			return handler.RespondEphemeral(e, "✗ Database insertion error: "+err.Error())
		}

		// Public action buttons (🎉 for entry count and 👥 for participants) with final DB ID
		enterBtn := discord.ButtonComponent{
			Style:    discord.ButtonStylePrimary,
			Label:    "0",
			CustomID: fmt.Sprintf("giveaway_enter_btn:%d", gw.ID),
			Emoji:    &discord.ComponentEmoji{Name: "🎉"},
		}
		partsBtn := discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    "Participants",
			CustomID: fmt.Sprintf("giveaway_participants_btn:%d", gw.ID),
			Emoji:    &discord.ComponentEmoji{Name: "👥"},
		}

		// Send message to the channel with the embed
		publicMsg, err := e.Client().Rest.CreateMessage(session.ChannelID, discord.NewMessageCreate().
			WithEmbeds(embed).
			WithComponents(discord.NewActionRow(enterBtn, partsBtn)),
		)
		if err != nil {
			// Clean up database if Discord message failed
			_ = queries.DeleteGiveaway(ctx, gw.ID)
			return handler.RespondEphemeral(e, "✗ Failed to launch giveaway message: "+err.Error())
		}

		// Update database entry with the actual message ID
		err = queries.UpdateGiveawayMessage(ctx, db.UpdateGiveawayMessageParams{
			MessageID: int64(publicMsg.ID),
			ID:        gw.ID,
		})
		if err != nil {
			// Delete public message and database entry to ensure absolute state consistency
			_ = e.Client().Rest.DeleteMessage(session.ChannelID, publicMsg.ID)
			_ = queries.DeleteGiveaway(ctx, gw.ID)
			return handler.RespondEphemeral(e, "✗ Failed to update message ID in database: "+err.Error())
		}

		// Clean session draft
		sm.Delete(*e.GuildID(), e.User().ID)

		// Schedule drawing job
		wm.ScheduleGiveaway(gw.ID, session.Duration)

		return e.UpdateMessage(discord.NewMessageUpdate().
			WithContent("✓ Giveaway successfully launched! 🚀").
			ClearEmbeds().
			ClearComponents(),
		)
	}
}

// HandleEnterBtn handles click-to-win entries with full gates validation.
func HandleEnterBtn(queries *db.Queries, wm *WorkerManager) handler.ButtonHandler {
	return func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		parts := strings.Split(e.Data.CustomID(), ":")
		if len(parts) != 2 {
			return nil
		}

		var giveawayID int64
		var giveaway db.Giveaway
		var err error

		giveawayID, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			// Fallback: search for giveaway using the Discord Message ID.
			// This provides a double-redundancy safety mechanism in case
			// the initial button custom IDs were not updated to the database ID.
			giveaway, err = queries.GetGiveawayByMessage(ctx, int64(e.Message.ID))
			if err != nil {
				return handler.RespondEphemeral(e, "✗ Invalid giveaway ID format and could not resolve giveaway by message.")
			}
			giveawayID = giveaway.ID
		} else {
			giveaway, err = queries.GetGiveaway(ctx, giveawayID)
			if err != nil {
				return handler.RespondEphemeral(e, "✗ Giveaway not found in database.")
			}
		}

		if giveaway.Ended || time.Now().After(giveaway.EndsAt) {
			// Trigger immediate drawing/ending if not marked ended in DB yet
			if !giveaway.Ended {
				go wm.DrawAndEndGiveaway(context.Background(), giveaway.ID)
			} else {
				// Proactively update message components to disable the button
				actionComponents := []discord.InteractiveComponent{
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
					discord.ButtonComponent{
						Style:    discord.ButtonStyleSecondary,
						Label:    "Reroll",
						CustomID: fmt.Sprintf("giveaway_reroll_btn:%d", giveaway.ID),
						Emoji:    &discord.ComponentEmoji{Name: "🔄"},
					},
				}
				_, _ = e.Client().Rest.UpdateMessage(e.Message.ChannelID, e.Message.ID, discord.NewMessageUpdate().
					WithComponents(discord.NewActionRow(actionComponents...)),
				)
			}

			return handler.RespondEphemeral(e, "✗ This giveaway has already ended!")
		}

		userID := int64(e.User().ID)

		// Check if already entered
		entered, err := queries.HasEntry(ctx, db.HasEntryParams{GiveawayID: giveawayID, UserID: userID})
		if err == nil && entered {
			return handler.RespondEphemeral(e, "✓ You are already entered in this giveaway! 🍀")
		}

		// GATING 1: Required Role
		if giveaway.RequiredRole.Valid {
			requiredRoleID := snowflake.ID(giveaway.RequiredRole.Int64)
			hasRole := false
			if e.Member() != nil {
				for _, rID := range e.Member().RoleIDs {
					if rID == requiredRoleID {
						hasRole = true
						break
					}
				}
			}
			if !hasRole {
				return handler.RespondEphemeral(e, fmt.Sprintf("✗ You do not have the required <@&%s> role to participate!", requiredRoleID.String()))
			}
		}

		// GATING 2: Minimum Level Requirement
		if giveaway.MinLevel > 0 {
			userStats, err := queries.GetUser(ctx, db.GetUserParams{
				UserId:  strconv.FormatInt(userID, 10),
				GuildId: int64(*e.GuildID()),
			})
			if err != nil || userStats.Level < giveaway.MinLevel {
				return handler.RespondEphemeral(e, fmt.Sprintf("✗ You must be at least **Level %d** to enter this giveaway! (Your current level: %d)", giveaway.MinLevel, userStats.Level))
			}
		}

		// GATING 3: Account Age Gating
		if giveaway.MinAccountDays > 0 {
			createdAt := e.User().ID.Time()
			accountAgeDays := int(time.Since(createdAt).Hours() / 24)
			if accountAgeDays < int(giveaway.MinAccountDays) {
				return handler.RespondEphemeral(e, fmt.Sprintf("✗ Your account must be at least **%d days old** to participate! (Your account age: %d days)", giveaway.MinAccountDays, accountAgeDays))
			}
		}

		// Insert Entry
		err = queries.InsertEntry(ctx, db.InsertEntryParams{
			GiveawayID: giveawayID,
			UserID:     userID,
		})
		if err != nil {
			return handler.RespondEphemeral(e, "✗ Failed to register entry. Please try again.")
		}

		// Count updated entries to display dynamically on the enter button label
		count, err := queries.CountEntries(ctx, giveawayID)
		if err != nil {
			count = 0
		}

		// Rebuild components with the updated entry count
		actionComponents := []discord.InteractiveComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStylePrimary,
				Label:    fmt.Sprintf("%d", count),
				CustomID: fmt.Sprintf("giveaway_enter_btn:%d", giveawayID),
				Emoji:    &discord.ComponentEmoji{Name: "🎉"},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    "Participants",
				CustomID: fmt.Sprintf("giveaway_participants_btn:%d", giveawayID),
				Emoji:    &discord.ComponentEmoji{Name: "👥"},
			},
		}

		_, _ = e.Client().Rest.UpdateMessage(e.Message.ChannelID, e.Message.ID, discord.NewMessageUpdate().
			WithComponents(discord.NewActionRow(actionComponents...)),
		)

		return handler.RespondEphemeral(e, fmt.Sprintf("✓ You have successfully entered the giveaway for **%s**! Good luck! 🍀", giveaway.Prize))
	}
}

// parseDuration provides an enhanced parser for duration inputs, adding full days ('d') unit support which Go's standard time.ParseDuration lacks.
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		days, err := strconv.ParseFloat(daysStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid days format")
		}
		return time.Duration(days * 24 * float64(time.Hour)), nil
	}
	return time.ParseDuration(s)
}

// HandleGiveawayEndSlash ends an active giveaway immediately and draws winners early.
func HandleGiveawayEndSlash(queries *db.Queries, wm *WorkerManager) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		data := e.SlashCommandInteractionData()
		msgIDStr := data.String("message_id")

		msgID, err := strconv.ParseInt(msgIDStr, 10, 64)
		if err != nil {
			return handler.RespondEphemeral(e, "✗ Invalid Message ID format.")
		}

		// Find active giveaway
		gw, err := queries.GetActiveGiveawayByMessage(ctx, msgID)
		if err != nil {
			return handler.RespondEphemeral(e, "✗ No active giveaway found with that Message ID.")
		}

		// Authorization Check: Must be the Host or have Manage Server permission
		isHost := snowflake.ID(gw.HostID) == e.User().ID
		isAdmin := false
		if e.Member() != nil {
			isAdmin = e.Member().Permissions.Has(discord.PermissionManageGuild) || e.Member().Permissions.Has(discord.PermissionAdministrator)
		}

		if !isHost && !isAdmin {
			return handler.RespondEphemeral(e, "✗ Only the giveaway host or server administrators can end giveaways early.")
		}

		// Draw winners immediately in the background
		go wm.DrawAndEndGiveaway(context.Background(), gw.ID)

		return handler.RespondEphemeral(e, fmt.Sprintf("✓ Giveaway for **%s** is ending immediately! Drawing winners...", gw.Prize))
	}
}

// HandleGiveawayRerollSlash rerolls an ended giveaway's winners.
func HandleGiveawayRerollSlash(queries *db.Queries, wm *WorkerManager) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		data := e.SlashCommandInteractionData()
		msgIDStr := data.String("message_id")

		msgID, err := strconv.ParseInt(msgIDStr, 10, 64)
		if err != nil {
			return handler.RespondEphemeral(e, "✗ Invalid Message ID format.")
		}

		// Get giveaway
		gw, err := queries.GetGiveawayByMessage(ctx, msgID)
		if err != nil {
			return handler.RespondEphemeral(e, "✗ No giveaway found with that Message ID.")
		}

		if !gw.Ended {
			return handler.RespondEphemeral(e, "✗ This giveaway has not ended yet. Use `/giveaway end` to end it early.")
		}

		// Authorization Check: Must be the Host or have Manage Server permission
		isHost := snowflake.ID(gw.HostID) == e.User().ID
		isAdmin := false
		if e.Member() != nil {
			isAdmin = e.Member().Permissions.Has(discord.PermissionManageGuild) || e.Member().Permissions.Has(discord.PermissionAdministrator)
		}

		winners, err := wm.DrawReroll(ctx, *e.GuildID(), e.User().ID, isHost, isAdmin, gw.ID)
		if err != nil {
			return handler.RespondEphemeral(e, fmt.Sprintf("✗ Failed to reroll: %v", err))
		}

		return handler.RespondEphemeral(e, fmt.Sprintf("✓ Successfully rerolled winners! New winner(s): %s", winners))
	}
}

// HandleGiveawayCancelSlash cancels an active giveaway, removing database entries and updates the post message.
func HandleGiveawayCancelSlash(queries *db.Queries, wm *WorkerManager) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		data := e.SlashCommandInteractionData()
		msgIDStr := data.String("message_id")

		msgID, err := strconv.ParseInt(msgIDStr, 10, 64)
		if err != nil {
			return handler.RespondEphemeral(e, "✗ Invalid Message ID format.")
		}

		// Get giveaway
		gw, err := queries.GetGiveawayByMessage(ctx, msgID)
		if err != nil {
			return handler.RespondEphemeral(e, "✗ No giveaway found with that Message ID.")
		}

		if gw.Ended {
			return handler.RespondEphemeral(e, "✗ This giveaway has already ended and cannot be cancelled.")
		}

		// Authorization Check: Must be the Host or have Manage Server permission
		isHost := snowflake.ID(gw.HostID) == e.User().ID
		isAdmin := false
		if e.Member() != nil {
			isAdmin = e.Member().Permissions.Has(discord.PermissionManageGuild) || e.Member().Permissions.Has(discord.PermissionAdministrator)
		}

		if !isHost && !isAdmin {
			return handler.RespondEphemeral(e, "✗ Only the giveaway host or server administrators can cancel giveaways.")
		}

		// Delete from database
		err = queries.DeleteEntriesForGiveaway(ctx, gw.ID)
		if err != nil {
			return handler.RespondEphemeral(e, "✗ Failed to delete giveaway entries from database.")
		}

		err = queries.DeleteGiveaway(ctx, gw.ID)
		if err != nil {
			return handler.RespondEphemeral(e, "✗ Failed to delete giveaway from database.")
		}

		// Update public message to "Cancelled"
		chID := snowflake.ID(gw.ChannelID)
		mID := snowflake.ID(gw.MessageID)

		_, _ = e.Client().Rest.CreateMessage(chID, discord.NewMessageCreate().
			WithContent(fmt.Sprintf("🛑 The giveaway for **%s** has been cancelled by an administrator.", gw.Prize)).
			WithMessageReference(&discord.MessageReference{MessageID: &mID}),
		)

		_, _ = e.Client().Rest.UpdateMessage(chID, mID, discord.MessageUpdate{
			Embeds: &[]discord.Embed{
				{
					Title:       "🛑 GIVEAWAY CANCELLED",
					Description: fmt.Sprintf("The giveaway for **%s** has been cancelled.", gw.Prize),
					Color:       0xcc0000,
					Footer: &discord.EmbedFooter{
						Text: "Cancelled • f(x) Giveaways",
					},
				},
			},
			Components: &[]discord.LayoutComponent{},
		})

		return handler.RespondEphemeral(e, "✓ Giveaway has been successfully cancelled and deleted.")
	}
}

// HandleParticipantsBtn lists all entries for the giveaway.
func HandleParticipantsBtn(queries *db.Queries) handler.ButtonHandler {
	return func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		parts := strings.Split(e.Data.CustomID(), ":")
		if len(parts) != 2 {
			return nil
		}

		var giveawayID int64
		var giveaway db.Giveaway
		var err error

		giveawayID, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			// Fallback: search for giveaway using the Discord Message ID.
			// This provides a double-redundancy safety mechanism in case
			// the initial button custom IDs were not updated to the database ID.
			giveaway, err = queries.GetGiveawayByMessage(ctx, int64(e.Message.ID))
			if err != nil {
				return handler.RespondEphemeral(e, "✗ Invalid giveaway ID format and could not resolve giveaway by message.")
			}
			giveawayID = giveaway.ID
		} else {
			giveaway, err = queries.GetGiveaway(ctx, giveawayID)
			if err != nil {
				return handler.RespondEphemeral(e, "✗ Giveaway not found in database.")
			}
		}

		entries, err := queries.ListEntries(ctx, giveawayID)
		if err != nil {
			return handler.RespondEphemeral(e, "✗ Failed to fetch participants list.")
		}

		if len(entries) == 0 {
			return handler.RespondEphemeral(e, "ℹ️ No entries in this giveaway yet. Be the first!")
		}

		var mentions []string
		for _, uID := range entries {
			mentions = append(mentions, fmt.Sprintf("<@%d>", uID))
		}

		// Pagination / Capping: Discord ephemeral limit is 2000 chars, let's limit lists to first 50 mentions
		count := len(mentions)
		displayList := mentions
		cappingMsg := ""
		if count > 50 {
			displayList = mentions[:50]
			cappingMsg = fmt.Sprintf("\n\n*...and %d more participants*", count-50)
		}

		embed := discord.Embed{
			Title:       fmt.Sprintf("👥 **Participants for %s** (%d total)", giveaway.Prize, count),
			Description: strings.Join(displayList, ", ") + cappingMsg,
			Color:       0x3498db,
		}

		return e.CreateMessage(discord.NewMessageCreate().
			WithEmbeds(embed).
			WithFlags(discord.MessageFlagEphemeral),
		)
	}
}
