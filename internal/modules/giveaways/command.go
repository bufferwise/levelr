package giveaways

import "github.com/disgoorg/disgo/discord"

var GiveawayCommand = discord.SlashCommandCreate{
	Name:        "giveaway",
	Description: "Manage server giveaways",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionSubCommand{
			Name:        "create",
			Description: "Start the interactive giveaway setup wizard",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "end",
			Description: "Force-end an active giveaway early and draw winners",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "message_id",
					Description: "Message ID of the target giveaway",
					Required:    true,
				},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "reroll",
			Description: "Reroll a new winner from an ended giveaway's entrants",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "message_id",
					Description: "Message ID of the target giveaway",
					Required:    true,
				},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "cancel",
			Description: "Cancel an active giveaway and delete its message",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "message_id",
					Description: "Message ID of the target giveaway",
					Required:    true,
				},
			},
		},
	},
}
