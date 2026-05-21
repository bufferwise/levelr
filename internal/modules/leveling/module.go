package leveling

import (
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"

	"github.com/bufferwise/levelr/internal/cache"
	"github.com/bufferwise/levelr/internal/config"
	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/handler"
	"github.com/bufferwise/levelr/internal/module"
	"github.com/bufferwise/levelr/internal/services"
)

// Mod is the leveling module.
type Mod struct {
	cfg     *config.Config
	queries *db.Queries
	client  *bot.Client
	cache   *cache.Client
	blSvc   *services.BlacklistService
	multSvc *services.MultiplierService
	xpSvc   *XPService
}

func New(cfg *config.Config, queries *db.Queries, client *bot.Client, cacheClient *cache.Client, blSvc *services.BlacklistService, multSvc *services.MultiplierService, notify services.Notifier) *Mod {
	xpSvc := NewXPService(queries, client, notify)
	return &Mod{
		cfg:     cfg,
		queries: queries,
		client:  client,
		cache:   cacheClient,
		blSvc:   blSvc,
		multSvc: multSvc,
		xpSvc:   xpSvc,
	}
}

func (m *Mod) Name() string { return "leveling" }

func (m *Mod) RegisterCommands(router *handler.Router) {
	router.Command("rank", HandleRank(m.queries, m.blSvc))
	router.Command("leaderboard", HandleLeaderboard(m.queries))
	router.SubCommand("admin", "setlevel", HandleSetLevel(m.queries))

	// Register pagination button handler
	router.Button("leaderboard:", HandleLeaderboardButton(m.queries))
}

func (m *Mod) RegisterListeners() []bot.EventListener {
	return []bot.EventListener{
		MessageListener(m.cfg, m.blSvc, m.multSvc, m.xpSvc),
		VoiceListener(m.cfg, m.cache),
	}
}

func (m *Mod) Workers() []module.Worker {
	return []module.Worker{
		{
			Name: "voice_ticker",
			Run:  StartVoiceTicker(m.cfg, m.client, m.cache, m.blSvc, m.multSvc, m.xpSvc),
		},
		{
			Name: "weekly_report",
			Run:  StartWeeklyReportWorker(m.queries, m.xpSvc.Notify, m.cfg.MainGuildID),
		},
		{
			Name: "blacklist_expiration",
			Run:  StartBlacklistExpirationWorker(m.blSvc),
		},
	}
}

func (m *Mod) SlashCommands() []discord.ApplicationCommandCreate {
	return []discord.ApplicationCommandCreate{
		RankCommand,
		LeaderboardCommand,
	}
}

func (m *Mod) AdminSubCommands() []module.AdminSubDef {
	return []module.AdminSubDef{
		{SubCommand: SetLevelSubCommand},
	}
}
