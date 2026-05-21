package sidekicks

import (
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"

	"github.com/bufferwise/levelr/internal/config"
	"github.com/bufferwise/levelr/internal/handler"
	"github.com/bufferwise/levelr/internal/module"
)

// Mod defines the sidekicks module for stateless utility features.
type Mod struct {
	cfg    *config.Config
	client *bot.Client
}

// New constructs the sidekicks module mapping dependencies structurally.
func New(cfg *config.Config, client *bot.Client) *Mod {
	return &Mod{
		cfg:    cfg,
		client: client,
	}
}

func (m *Mod) Name() string {
	return "sidekicks"
}

func (m *Mod) RegisterCommands(router *handler.Router) {
	// Interaction handlers
	router.Button("sidekick:selfrole:toggle:", m.HandleSelfRoleToggle)
	router.Button("sidekick:robloxrole:toggle:", m.HandleRobloxRoleToggle)
	router.Button("sidekick:apexspider:yearly_rewind", m.HandleYearlyRewindButton)
	router.Button("sidekick:apexspider:primespiders:page:", m.HandlePrimeSpidersPageButton)
	router.Button("sidekick:apexspider:primespiders", m.HandlePrimeSpidersButton)
}

func (m *Mod) RegisterListeners() []bot.EventListener {
	return nil
}

func (m *Mod) Workers() []module.Worker {
	return []module.Worker{
		{
			Name: "selfrole_persistence_check",
			Run:  m.CheckAndSendSelfRole,
		},
		{
			Name: "robloxrole_persistence_check",
			Run:  m.CheckAndSendRobloxRole,
		},
		{
			Name: "apexspider_persistence_check",
			Run:  m.CheckAndSendApexSpider,
		},
	}
}

func (m *Mod) SlashCommands() []discord.ApplicationCommandCreate {
	return nil
}
