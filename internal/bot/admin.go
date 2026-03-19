package bot

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/omit"

	"github.com/bufferwise/levelr/internal/module"
)

// BuildAdminCommand assembles the /admin command from core subcommand groups
// plus module contributions.
func BuildAdminCommand(moduleContribs []module.AdminSubDef) discord.SlashCommandCreate {
	options := []discord.ApplicationCommandOption{
		BlacklistSubGroup,
		MultiplierSubGroup,
	}

	// Add module contributions (e.g., setlevel from leveling, drop from drops)
	for _, contrib := range moduleContribs {
		options = append(options, contrib.SubCommand)
	}

	return discord.SlashCommandCreate{
		Name:                     "admin",
		Description:              "∂ Administrative commands for Function",
		DefaultMemberPermissions: omit.New(Ptr(discord.Permissions(discord.PermissionManageGuild))),
		Options:                  options,
	}
}
