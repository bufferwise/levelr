package presence

import (
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"

	"github.com/bufferwise/levelr/internal/config"
	"github.com/bufferwise/levelr/internal/handler"
	"github.com/bufferwise/levelr/internal/module"
)

// Mod is the presence module.
type Mod struct {
	cfg     *config.Config
	client  *bot.Client
	presSvc *PresenceService
}

func New(cfg *config.Config, client *bot.Client) *Mod {
	presSvc := NewPresenceService(cfg, client)
	return &Mod{cfg: cfg, client: client, presSvc: presSvc}
}

func (m *Mod) Name() string { return "presence" }

func (m *Mod) RegisterCommands(router *handler.Router) {}

func (m *Mod) RegisterListeners() []bot.EventListener {
	return nil
}

func (m *Mod) Workers() []module.Worker {
	if !m.cfg.YoutubePresence {
		return nil
	}
	return []module.Worker{
		{
			Name: "youtube_presence_ticker",
			Run:  m.presSvc.StartTicker,
		},
	}
}

func (m *Mod) SlashCommands() []discord.ApplicationCommandCreate {
	return nil
}
