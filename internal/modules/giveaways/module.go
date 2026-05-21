package giveaways

import (
	"context"
	"strconv"
	"strings"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/bufferwise/levelr/internal/config"
	db "github.com/bufferwise/levelr/internal/db/sqlc"
	"github.com/bufferwise/levelr/internal/handler"
	"github.com/bufferwise/levelr/internal/module"
)

type Mod struct {
	cfg      *config.Config
	queries  *db.Queries
	client   *bot.Client
	sessions *SessionManager
	worker   *WorkerManager
}

func New(cfg *config.Config, queries *db.Queries, client *bot.Client) *Mod {
	return &Mod{
		cfg:      cfg,
		queries:  queries,
		client:   client,
		sessions: NewSessionManager(),
		worker:   NewWorkerManager(client, queries),
	}
}

func (m *Mod) Name() string { return "giveaways" }

func (m *Mod) RegisterCommands(router *handler.Router) {
	// Slash Command Router mapping
	router.SubCommand("giveaway", "create", HandleGiveawayCreateSlash(m.sessions))
	router.SubCommand("giveaway", "end", HandleGiveawayEndSlash(m.queries, m.worker))
	router.SubCommand("giveaway", "reroll", HandleGiveawayRerollSlash(m.queries, m.worker))
	router.SubCommand("giveaway", "cancel", HandleGiveawayCancelSlash(m.queries, m.worker))

	// Ephemeral Setup Control Panel Buttons
	router.Button("giveaway_set_level_btn", HandleLevelBtn(m.sessions))
	router.Button("giveaway_set_min_age_btn", HandleMinAgeBtn(m.sessions))
	router.Button("giveaway_set_mults_btn", HandleMultsBtn(m.sessions))
	router.Button("giveaway_launch_btn", HandleLaunchBtn(m.queries, m.sessions, m.worker))
	router.Button("giveaway_cancel_btn", HandleCancelBtn(m.sessions))

	// Public Gating Buttons
	router.Button("giveaway_enter_btn:", HandleEnterBtn(m.queries, m.worker))
	router.Button("giveaway_participants_btn:", HandleParticipantsBtn(m.queries))

	// Reroll action
	router.Button("giveaway_reroll_btn:", func(ctx context.Context, e *events.ComponentInteractionCreate) error {
		parts := strings.Split(e.Data.CustomID(), ":")
		if len(parts) != 2 {
			return nil
		}
		giveawayID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return handler.RespondEphemeral(e, "✗ Invalid giveaway ID format for reroll.")
		}
		return m.worker.PerformReroll(ctx, e, giveawayID)
	})

	// Role Dropdown Setup
	router.Select("giveaway_role_select", HandleRoleSelect(m.sessions))

	// Modals Input Routing
	router.Modal("giveaway_create_modal", HandleCreateModal(m.sessions))
	router.Modal("giveaway_level_req_modal", HandleLevelModal(m.sessions))
	router.Modal("giveaway_min_age_modal", HandleMinAgeModal(m.sessions))
	router.Modal("giveaway_multiplier_modal", HandleMultsModal(m.sessions))
}

func (m *Mod) RegisterListeners() []bot.EventListener {
	return nil
}

func (m *Mod) Workers() []module.Worker {
	return []module.Worker{
		{
			Name: "giveaways_recovery",
			Run:  m.worker.StartRecoveryWorker,
		},
	}
}

func (m *Mod) SlashCommands() []discord.ApplicationCommandCreate {
	return []discord.ApplicationCommandCreate{
		GiveawayCommand,
	}
}
