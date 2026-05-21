package sidekicks

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"log/slog"
	"strings"
)

// RoleOption defines the hardcoded configuration for a self-role option.
type RoleOption struct {
	Label       string
	ButtonLabel string
	RoleID      uint64
	Emoji       string
}

func Ptr[T any](v T) *T {
	return &v
}

// Hardcoded roles configuration (Update these with actual IDs)
var selfRoles = []RoleOption{
	{Label: "Update Ping", ButtonLabel: "Updates", RoleID: 1412085219355398235, Emoji: "<:cal2:1252241754183434281>"},
	{Label: "Event Ping", ButtonLabel: "Events", RoleID: 1412083980811112580, Emoji: "<:handshake:1252241509894324256>"},
	{Label: "Fun Ping", ButtonLabel: "Fun", RoleID: 1416410559632375931, Emoji: "<:senpai_wow:1357014819848585427>"},
	{Label: "Giveaway Ping", ButtonLabel: "Giveaways", RoleID: 1412085101764022366, Emoji: "<:Spider_Diamond:994139955758641165>"},
	{Label: "Social feeds Ping", ButtonLabel: "Socials", RoleID: 1412082916049293424, Emoji: "<:Spider_blue_verified:987620618960769064>"},
}

// HandleSelfRoleToggle processes a button toggle interaction.
func (m *Mod) HandleSelfRoleToggle(ctx context.Context, e *events.ComponentInteractionCreate) error {
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
	for _, opt := range selfRoles {
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
			slog.Error("failed to remove role", slog.Any("err", err), slog.Uint64("role_id", uint64(roleID)))
			content = fmt.Sprintf("Failed to remove **%s** role: %s", matchedRole.Label, err.Error())
		}
	} else {
		err = m.client.Rest.AddMemberRole(*e.GuildID(), e.User().ID, roleID)
		if err == nil {
			content = fmt.Sprintf("Added the **%s** role.", matchedRole.Label)
		} else {
			slog.Error("failed to add role", slog.Any("err", err), slog.Uint64("role_id", uint64(roleID)))
			content = fmt.Sprintf("Failed to add **%s** role: %s", matchedRole.Label, err.Error())
		}
	}

	return e.CreateMessage(discord.MessageCreate{
		Content: content,
		Flags:   discord.MessageFlagEphemeral,
	})
}

// buildSelfRoleMessage constructs the standard self-role message structure.
func (m *Mod) buildSelfRoleMessage() discord.MessageCreate {
	bannerURL := "https://media.discordapp.net/attachments/1411726124844711946/1419536818407477319/standard_3.gif"

	var sb strings.Builder
	for _, opt := range selfRoles {
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
			Content: "# <:Spider_Diamond:994139955758641165> __Self Roles__ :",
		},
		discord.NewSmallSeparator(),
		discord.TextDisplayComponent{
			Content: sb.String(),
		},
		discord.NewSmallSeparator(),
		discord.TextDisplayComponent{
			Content: "> Click the buttons below to toggle pings for giveaways and updates.",
		},
	)
	container.AccentColor = 0xf1c40f

	var buttons []discord.InteractiveComponent
	for _, opt := range selfRoles {
		emojiName, emojiID, emojiAnimated := parseEmoji(opt.Emoji)
		buttons = append(buttons, discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    opt.ButtonLabel,
			CustomID: fmt.Sprintf("sidekick:selfrole:toggle:%d", opt.RoleID),
			Emoji: &discord.ComponentEmoji{
				Name:     emojiName,
				ID:       emojiID,
				Animated: emojiAnimated,
			},
		})
	}

	return discord.NewMessageCreateV2(container, discord.NewActionRow(buttons...))
}

// CheckAndSendSelfRole checks if the menu already exists in the target channel and sends it if not.
func (m *Mod) CheckAndSendSelfRole(ctx context.Context) {
	channelID := snowflake.ID(1411738885641212125)

	// Search for existing message in the last 50 messages
	messages, err := m.client.Rest.GetMessages(channelID, 0, 0, 0, 50)
	if err != nil {
		slog.Error("failed to get messages for self-role check", slog.Any("err", err), slog.Uint64("channel_id", uint64(channelID)))
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
							if strings.HasPrefix(button.CustomID, "sidekick:selfrole:toggle:") {
								exists = true
								existingMsgID = msg.ID
								break
							}
						} else if selectMenu, ok := component.(discord.StringSelectMenuComponent); ok {
							if selectMenu.CustomID == "sidekick:selfrole:menu" {
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
		slog.Info("self-role menu not found, sending new message", slog.Uint64("channel_id", uint64(channelID)))
		_, err = m.client.Rest.CreateMessage(channelID, m.buildSelfRoleMessage())
		if err != nil {
			slog.Error("failed to send self-role menu", slog.Any("err", err))
		}
	} else {
		slog.Info("self-role menu already exists, updating to ensure latest config", slog.Uint64("channel_id", uint64(channelID)), slog.Uint64("message_id", uint64(existingMsgID)))
		newContent := m.buildSelfRoleMessage()
		flags := discord.MessageFlags(32768)
		emptyEmbeds := []discord.Embed{}
		_, err = m.client.Rest.UpdateMessage(channelID, existingMsgID, discord.MessageUpdate{
			Content:    Ptr(""),
			Embeds:     &emptyEmbeds,
			Components: &newContent.Components,
			Flags:      &flags,
		})
		if err != nil {
			slog.Error("failed to update self-role menu", slog.Any("err", err))
		}
	}
}

// parseEmoji is a helper to manually parse Discord emoji strings since discord.ParseEmoji was not found.
func parseEmoji(s string) (name string, id snowflake.ID, animated bool) {
	if s == "" {
		return
	}
	if !strings.HasPrefix(s, "<") || !strings.HasSuffix(s, ">") {
		name = s // standard emoji
		return
	}

	// Custom emoji: <:name:id> or <a:name:id>
	content := s[1 : len(s)-1]
	if strings.HasPrefix(content, "a:") {
		animated = true
		content = content[2:]
	} else if strings.HasPrefix(content, ":") {
		content = content[1:]
	}

	parts := strings.Split(content, ":")
	if len(parts) == 2 {
		name = parts[0]
		id, _ = snowflake.Parse(parts[1])
	}
	return
}
