package drops

import (
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"

	"github.com/bufferwise/levelr/internal/config"
	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/handler"
	"github.com/bufferwise/levelr/internal/module"
)

// Mod is the drops module.
type Mod struct {
	cfg     *config.Config
	queries *db.Queries
	client  *bot.Client
	dropSvc *DropService
}

func New(cfg *config.Config, queries *db.Queries, client *bot.Client) *Mod {
	dropSvc := NewDropService(queries, client, cfg)
	return &Mod{cfg: cfg, queries: queries, client: client, dropSvc: dropSvc}
}

func (m *Mod) Name() string { return "drops" }

func (m *Mod) RegisterCommands(router *handler.Router) {
	router.Button("drop_", HandleDropButton(m.dropSvc))
	router.SubCommand("admin", "drop", HandleAdminDrop(m.queries, m.dropSvc))
}

func (m *Mod) RegisterListeners() []bot.EventListener {
	return nil
}

func (m *Mod) Workers() []module.Worker {
	return []module.Worker{
		{
			Name: "drop_ticker",
			Run:  StartDropTicker(m.cfg, m.dropSvc),
		},
	}
}

func (m *Mod) SlashCommands() []discord.ApplicationCommandCreate {
	return nil // No top-level commands, only admin subcommands
}

func (m *Mod) AdminSubCommands() []module.AdminSubDef {
	return []module.AdminSubDef{
		{SubCommand: DropSubCommand},
	}
}
