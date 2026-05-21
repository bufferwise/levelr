package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/handler"
	"github.com/bufferwise/levelr/internal/services"
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
				discord.ApplicationCommandOptionString{
					Name:        "reason",
					Description: "Reason for the blacklist",
					Required:    false,
				},
				discord.ApplicationCommandOptionString{
					Name:        "duration",
					Description: "Duration of the blacklist (e.g. 7d, 2w, 12h, 30m)",
					Required:    false,
				},
				discord.ApplicationCommandOptionBool{
					Name:        "hidden",
					Description: "Whether the blacklist should be hidden (Shadow Realm)",
					Required:    false,
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
				discord.ApplicationCommandOptionString{
					Name:        "reason",
					Description: "Reason for removal",
					Required:    false,
				},
			},
		},
		{
			Name:        "list",
			Description: "List active blacklisted entities",
		},
		{
			Name:        "logs",
			Description: "View recent blacklist audit logs",
		},
	},
}

func HandleBlacklistAdd(blSvc *services.BlacklistService) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		data := e.SlashCommandInteractionData()
		eType := data.String("type")
		idStr := CleanDiscordID(data.String("id"))

		reason := "Violation of server rules"
		if r, ok := data.OptString("reason"); ok {
			reason = r
		}

		var expiresAt *time.Time
		if durStr, ok := data.OptString("duration"); ok {
			dur, err := services.ParseCustomDuration(durStr)
			if err != nil {
				return handler.RespondEphemeral(e, "❌ **Invalid Duration:** "+err.Error())
			}
			t := time.Now().Add(dur)
			expiresAt = &t
		}

		isHidden := false
		if h, ok := data.OptBool("hidden"); ok {
			isHidden = h
		}

		guildIDRaw := e.GuildID()
		if guildIDRaw == nil {
			return handler.RespondEphemeral(e, "❌ This command can only be used in a server.")
		}
		guildIDStr := strconv.FormatUint(uint64(*guildIDRaw), 10)
		actorIDStr := e.User().ID.String()

		err := blSvc.AddBlacklist(ctx, guildIDStr, eType, idStr, reason, actorIDStr, isHidden, expiresAt, actorIDStr)
		if err != nil {
			return handler.RespondEphemeral(e, "❌ **Database Error:** "+err.Error())
		}

		expiresDesc := "permanently"
		if expiresAt != nil {
			expiresDesc = fmt.Sprintf("until <t:%d:R>", expiresAt.Unix())
		}
		var shadowDesc string
		if isHidden {
			shadowDesc = " (hidden in rank)"
		}

		embed := discord.Embed{
			Title: "🛡️ Blacklist Updated",
			Description: fmt.Sprintf(
				"I've added the **%s** `%s` to the blacklist %s because of: *%s*%s.",
				strings.Title(eType), idStr, expiresDesc, reason, shadowDesc,
			),
			Color: 0xE06C75,
		}

		// Log admin action to Discord log channel
		actorMention := fmt.Sprintf("<@%s>", actorIDStr)
		var targetMention string
		if eType == "user" {
			targetMention = fmt.Sprintf("<@%s>", idStr)
		} else if eType == "channel" {
			targetMention = fmt.Sprintf("<#%s>", idStr)
		} else {
			targetMention = fmt.Sprintf("<@&%s>", idStr)
		}
		now := time.Now()
		adminLogEmbed := discord.Embed{
			Title: "🛡️ Blacklist Added",
			Description: fmt.Sprintf(
				"**Actor:** %s\n**Target:** %s (`%s`)\n**Type:** %s\n**Reason:** *%s*\n**Duration:** %s%s",
				actorMention, targetMention, idStr, strings.Title(eType), reason, expiresDesc, shadowDesc,
			),
			Color: 0xE06C75,
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

		return e.CreateMessage(discord.NewMessageCreate().
			WithEmbeds(embed).
			WithFlags(discord.MessageFlagEphemeral))
	}
}

func HandleBlacklistRemove(blSvc *services.BlacklistService) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		data := e.SlashCommandInteractionData()
		eType := data.String("type")
		idStr := CleanDiscordID(data.String("id"))

		reason := "Administrative removal"
		if r, ok := data.OptString("reason"); ok {
			reason = r
		}

		guildIDRaw := e.GuildID()
		if guildIDRaw == nil {
			return handler.RespondEphemeral(e, "❌ This command can only be used in a server.")
		}
		guildIDStr := strconv.FormatUint(uint64(*guildIDRaw), 10)
		actorIDStr := e.User().ID.String()

		err := blSvc.RemoveBlacklist(ctx, guildIDStr, eType, idStr, actorIDStr, reason)
		if err != nil {
			return handler.RespondEphemeral(e, "❌ **Database Error:** "+err.Error())
		}

		embed := discord.Embed{
			Title: "✅ Blacklist Updated",
			Description: fmt.Sprintf(
				"I've removed the **%s** `%s` from the blacklist.",
				strings.Title(eType), idStr,
			),
			Color: 0xA3BE8C,
		}

		// Log admin action to Discord log channel
		actorMention := fmt.Sprintf("<@%s>", actorIDStr)
		var targetMention string
		if eType == "user" {
			targetMention = fmt.Sprintf("<@%s>", idStr)
		} else if eType == "channel" {
			targetMention = fmt.Sprintf("<#%s>", idStr)
		} else {
			targetMention = fmt.Sprintf("<@&%s>", idStr)
		}
		now := time.Now()
		adminLogEmbed := discord.Embed{
			Title: "✅ Blacklist Removed",
			Description: fmt.Sprintf(
				"**Actor:** %s\n**Target:** %s (`%s`)\n**Type:** %s\n**Reason:** *%s*",
				actorMention, targetMention, idStr, strings.Title(eType), reason,
			),
			Color: 0xA3BE8C,
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

		return e.CreateMessage(discord.NewMessageCreate().
			WithEmbeds(embed).
			WithFlags(discord.MessageFlagEphemeral))
	}
}

func HandleBlacklistList(blSvc *services.BlacklistService) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		guildIDRaw := e.GuildID()
		if guildIDRaw == nil {
			return handler.RespondEphemeral(e, "❌ This command can only be used in a server.")
		}
		guildIDStr := strconv.FormatUint(uint64(*guildIDRaw), 10)

		embed, buttons := buildBlacklistListMessage(ctx, blSvc, guildIDStr, 0)

		msgCreate := discord.NewMessageCreate().
			WithEmbeds(embed).
			AddActionRow(buttons...).
			WithFlags(discord.MessageFlagEphemeral)

		err := e.CreateMessage(msgCreate)
		if err == nil {
			// Auto-disable pagination buttons after 60 seconds
			time.AfterFunc(60*time.Second, func() {
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

func HandleBlacklistListButton(blSvc *services.BlacklistService) handler.ButtonHandler {
	return func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		parts := strings.Split(e.Data.CustomID(), ":")
		if len(parts) != 3 {
			return nil
		}

		guildIDStr := parts[1]
		var page int
		fmt.Sscanf(parts[2], "%d", &page)

		embed, buttons := buildBlacklistListMessage(ctx, blSvc, guildIDStr, page)

		msgUpdate := discord.NewMessageUpdate().
			WithEmbeds(embed).
			WithComponents(discord.NewActionRow(buttons...))

		err := e.UpdateMessage(msgUpdate)
		if err == nil {
			time.AfterFunc(60*time.Second, func() {
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

func HandleBlacklistLogs(blSvc *services.BlacklistService) handler.CommandHandler {
	return func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error {
		guildIDRaw := e.GuildID()
		if guildIDRaw == nil {
			return handler.RespondEphemeral(e, "❌ This command can only be used in a server.")
		}
		guildIDStr := strconv.FormatUint(uint64(*guildIDRaw), 10)

		embed, buttons := buildBlacklistLogsMessage(ctx, blSvc, guildIDStr, 0)

		msgCreate := discord.NewMessageCreate().
			WithEmbeds(embed).
			AddActionRow(buttons...).
			WithFlags(discord.MessageFlagEphemeral)

		err := e.CreateMessage(msgCreate)
		if err == nil {
			time.AfterFunc(60*time.Second, func() {
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

func HandleBlacklistLogsButton(blSvc *services.BlacklistService) handler.ButtonHandler {
	return func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		parts := strings.Split(e.Data.CustomID(), ":")
		if len(parts) != 3 {
			return nil
		}

		guildIDStr := parts[1]
		var page int
		fmt.Sscanf(parts[2], "%d", &page)

		embed, buttons := buildBlacklistLogsMessage(ctx, blSvc, guildIDStr, page)

		msgUpdate := discord.NewMessageUpdate().
			WithEmbeds(embed).
			WithComponents(discord.NewActionRow(buttons...))

		err := e.UpdateMessage(msgUpdate)
		if err == nil {
			time.AfterFunc(60*time.Second, func() {
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

func buildBlacklistListMessage(ctx context.Context, blSvc *services.BlacklistService, guildID string, page int) (discord.Embed, []discord.InteractiveComponent) {
	limit := int64(10)
	offset := int64(page) * limit

	rows, _ := blSvc.Queries().ListGuildBlacklist(ctx, db.ListGuildBlacklistParams{
		GuildId:   guildID,
		OffsetVal: offset,
		LimitVal:  limit,
	})

	var desc string
	if len(rows) == 0 {
		desc = "There are no blacklisted entities in this server."
	} else {
		var listText strings.Builder
		for i, r := range rows {
			expiresStr := "permanently"
			if r.ExpiresAt.Valid {
				expiresStr = fmt.Sprintf("until <t:%d:R>", r.ExpiresAt.Time.Unix())
			}
			hiddenStr := ""
			if r.IsHidden {
				hiddenStr = " (hidden)"
			}
			listText.WriteString(fmt.Sprintf(
				"%d. **%s `%s`**%s — reason: *%s*, added by <@%s>, %s\n",
				int(offset)+i+1,
				strings.Title(r.EntityType),
				r.EntityID,
				hiddenStr,
				r.Reason,
				r.AddedBy,
				expiresStr,
			))
		}
		desc = listText.String()
	}

	embed := discord.Embed{
		Title:       "📋 Blacklisted Entities",
		Description: desc,
		Color:       0xE06C75,
		Footer: &discord.EmbedFooter{
			Text: "Page " + strconv.Itoa(page+1),
		},
	}

	buttons := []discord.InteractiveComponent{
		discord.NewSecondaryButton("◀ Prev", fmt.Sprintf("blacklist_list:%s:%d", guildID, page-1)).WithDisabled(page <= 0),
		discord.NewSecondaryButton("Next ▶", fmt.Sprintf("blacklist_list:%s:%d", guildID, page+1)).WithDisabled(len(rows) < int(limit)),
	}

	return embed, buttons
}

func buildBlacklistLogsMessage(ctx context.Context, blSvc *services.BlacklistService, guildID string, page int) (discord.Embed, []discord.InteractiveComponent) {
	limit := int64(10)
	offset := int64(page) * limit

	rows, _ := blSvc.Queries().GetAuditLogs(ctx, db.GetAuditLogsParams{
		GuildId:   guildID,
		OffsetVal: offset,
		LimitVal:  limit,
	})

	var desc string
	if len(rows) == 0 {
		desc = "No audit logs found for this server."
	} else {
		var listText strings.Builder
		for i, r := range rows {
			reason := "No reason provided"
			if r.Reason.Valid && r.Reason.String != "" {
				reason = r.Reason.String
			}
			actionText := "Added"
			if r.Action == "REMOVED" {
				actionText = "Removed"
			} else if r.Action == "EXPIRED" {
				actionText = "Expired"
			}
			listText.WriteString(fmt.Sprintf(
				"%d. **%s**: %s `%s` by <@%s> — reason: *%s* (<t:%d:R>)\n",
				int(offset)+i+1,
				actionText,
				strings.Title(r.EntityType),
				r.EntityID,
				r.ActorID,
				reason,
				r.CreatedAt.Unix(),
			))
		}
		desc = listText.String()
	}

	embed := discord.Embed{
		Title:       "📜 Blacklist Audit Logs",
		Description: desc,
		Color:       0xE06C75,
		Footer: &discord.EmbedFooter{
			Text: "Page " + strconv.Itoa(page+1),
		},
	}

	buttons := []discord.InteractiveComponent{
		discord.NewSecondaryButton("◀ Prev", fmt.Sprintf("blacklist_logs:%s:%d", guildID, page-1)).WithDisabled(page <= 0),
		discord.NewSecondaryButton("Next ▶", fmt.Sprintf("blacklist_logs:%s:%d", guildID, page+1)).WithDisabled(len(rows) < int(limit)),
	}

	return embed, buttons
}

func CleanDiscordID(idStr string) string {
	idStr = strings.TrimSpace(idStr)
	if strings.HasPrefix(idStr, "<") && strings.HasSuffix(idStr, ">") {
		idStr = idStr[1 : len(idStr)-1]
		idStr = strings.TrimLeft(idStr, "@!&#")
	}
	return idStr
}
