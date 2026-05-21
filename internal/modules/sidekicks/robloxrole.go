package sidekicks

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

// Roblox roles configuration
var robloxRoles = []RoleOption{
	{Label: "Roblox Ping", ButtonLabel: "Roblox", RoleID: 1411947106976923658, Emoji: "<:roblox:1357065359634596040>"},
	{Label: "Blox Fruit Ping", ButtonLabel: "Blox Fruit", RoleID: 1411947164619509871, Emoji: "<:blox_fruit:1403611923249107036>"},
	{Label: "Steal A Brainrot Ping", ButtonLabel: "Steal A Brainrot", RoleID: 1411947990020657152, Emoji: "<:stealabrainrot:1412852917815345182>"},
	{Label: "Grow A Garden Ping", ButtonLabel: "Grow A Garden", RoleID: 1411948071243354195, Emoji: "<:grow_a_garden:1403612054337622026>"},
}

// HandleRobloxRoleToggle processes a button toggle interaction for Roblox roles.
func (m *Mod) HandleRobloxRoleToggle(ctx context.Context, e *events.ComponentInteractionCreate) error {
	customID := e.Data.CustomID()
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		return nil
	}
	roleIDStr := parts[3]
	roleID, err := snowflake.Parse(roleIDStr)
	if err != nil {
		return err
	}

	var matchedRole RoleOption
	found := false
	for _, opt := range robloxRoles {
		if opt.RoleID == uint64(roleID) {
			matchedRole = opt
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	userRoleIDs := e.Member().RoleIDs
	hasRole := false
	for _, rid := range userRoleIDs {
		if rid == roleID {
			hasRole = true
			break
		}
	}

	var content string
	if hasRole {
		err = m.client.Rest.RemoveMemberRole(*e.GuildID(), e.User().ID, roleID)
		if err == nil {
			content = fmt.Sprintf("Removed the **%s** role.", matchedRole.Label)
		} else {
			slog.Error("failed to remove roblox role", slog.Any("err", err), slog.Uint64("role_id", uint64(roleID)))
			content = fmt.Sprintf("Failed to remove **%s** role: %s", matchedRole.Label, err.Error())
		}
	} else {
		err = m.client.Rest.AddMemberRole(*e.GuildID(), e.User().ID, roleID)
		if err == nil {
			content = fmt.Sprintf("Added the **%s** role.", matchedRole.Label)
		} else {
			slog.Error("failed to add roblox role", slog.Any("err", err), slog.Uint64("role_id", uint64(roleID)))
			content = fmt.Sprintf("Failed to add **%s** role: %s", matchedRole.Label, err.Error())
		}
	}

	return e.CreateMessage(discord.MessageCreate{
		Content: content,
		Flags:   discord.MessageFlagEphemeral,
	})
}

// buildRobloxRoleMessage constructs the Roblox roles message structure.
func (m *Mod) buildRobloxRoleMessage() discord.MessageCreate {
	bannerURL := "https://media.discordapp.net/attachments/1411726124844711946/1419536818407477319/standard_3.gif"

	var sb strings.Builder
	for _, opt := range robloxRoles {
		sb.WriteString(fmt.Sprintf("%s **%s**\n", opt.Emoji, opt.Label))
	}

	container := discord.NewContainer(
		discord.MediaGalleryComponent{
			Items: []discord.MediaGalleryItem{
				{
					Media: discord.UnfurledMediaItem{
						URL: bannerURL,
					},
				},
			},
		},
		discord.NewSmallSeparator(),
		discord.TextDisplayComponent{
			Content: "# <:roblox:1357065359634596040> __Roblox Roles__ :",
		},
		discord.NewSmallSeparator(),
		discord.TextDisplayComponent{
			Content: sb.String(),
		},
		discord.NewSmallSeparator(),
		discord.TextDisplayComponent{
			Content: "> Click the buttons below to toggle pings for Roblox games.",
		},
	)
	container.AccentColor = 0x00A2FF // Classic Roblox Blue

	var buttons []discord.InteractiveComponent
	for _, opt := range robloxRoles {
		emojiName, emojiID, emojiAnimated := parseEmoji(opt.Emoji)
		buttons = append(buttons, discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    opt.ButtonLabel,
			CustomID: fmt.Sprintf("sidekick:robloxrole:toggle:%d", opt.RoleID),
			Emoji: &discord.ComponentEmoji{
				Name:     emojiName,
				ID:       emojiID,
				Animated: emojiAnimated,
			},
		})
	}

	return discord.NewMessageCreateV2(container, discord.NewActionRow(buttons...))
}

// CheckAndSendRobloxRole checks if the menu already exists in the target channel and sends it if not.
func (m *Mod) CheckAndSendRobloxRole(ctx context.Context) {
	channelID := snowflake.ID(1411946812582920192)

	// Search for existing message in the last 50 messages
	messages, err := m.client.Rest.GetMessages(channelID, 0, 0, 0, 50)
	if err != nil {
		slog.Error("failed to get messages for roblox role check", slog.Any("err", err), slog.Uint64("channel_id", uint64(channelID)))
		return
	}

	var existingMsgID snowflake.ID
	exists := false
	for _, msg := range messages {
		// Check if it's from the bot and has our custom ID
		if msg.Author.ID == m.client.ApplicationID {
			for _, container := range msg.Components {
				if actionRow, ok := container.(discord.ActionRowComponent); ok {
					for _, component := range actionRow.Components {
						if button, ok := component.(discord.ButtonComponent); ok {
							if strings.HasPrefix(button.CustomID, "sidekick:robloxrole:toggle:") {
								exists = true
								existingMsgID = msg.ID
								break
							}
						}
					}
				}
				if exists {
					break
				}
			}
		}
		if exists {
			break
		}
	}

	if !exists {
		slog.Info("roblox role menu not found, sending new message", slog.Uint64("channel_id", uint64(channelID)))
		_, err = m.client.Rest.CreateMessage(channelID, m.buildRobloxRoleMessage())
		if err != nil {
			slog.Error("failed to send roblox role menu", slog.Any("err", err))
		}
	} else {
		slog.Info("roblox role menu already exists, updating to ensure latest config", slog.Uint64("channel_id", uint64(channelID)), slog.Uint64("message_id", uint64(existingMsgID)))
		newContent := m.buildRobloxRoleMessage()
		flags := discord.MessageFlags(32768)
		emptyEmbeds := []discord.Embed{}
		_, err = m.client.Rest.UpdateMessage(channelID, existingMsgID, discord.MessageUpdate{
			Content:    Ptr(""),
			Embeds:     &emptyEmbeds,
			Components: &newContent.Components,
			Flags:      &flags,
		})
		if err != nil {
			slog.Error("failed to update roblox role menu", slog.Any("err", err))
		}
	}
}
