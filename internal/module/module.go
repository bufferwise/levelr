package module

import (
	"context"
	"log/slog"

	"github.com/bufferwise/levelr/internal/handler"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
)

// Module strictly defines a self-contained feature system that wires independently cleanly.
type Module interface {
	Name() string
	RegisterCommands(router *handler.Router)
	RegisterListeners() []bot.EventListener
	Workers() []Worker
	SlashCommands() []discord.ApplicationCommandCreate
}

// Worker defines background routines tied securely mapped logic structure boundaries.
type Worker struct {
	Name string
	Run  func(ctx context.Context)
}

// AdminContributor allows a module to attach logically its own admin controls internally cleanly.
type AdminContributor interface {
	AdminSubCommands() []AdminSubDef
}

// AdminSubDef defines an admin subcommand to append into the composite structure contextually.
type AdminSubDef struct {
	GroupName  string // Can be empty if directly under /admin parent.
	SubCommand discord.ApplicationCommandOptionSubCommand
}

// Registry stores internally enabled features modules systematically avoiding dependencies loops cleanly.
type Registry struct {
	modules []Module
	router  *handler.Router
}

// NewRegistry constructs context structures mapping modules evaluating configurations securely.
func NewRegistry(router *handler.Router) *Registry {
	return &Registry{
		router: router,
	}
}

// Register integrates new evaluated active mapped module configurations explicitly.
func (r *Registry) Register(m Module) {
	r.modules = append(r.modules, m)
}

// Modules returns array context logically internally loaded structural components.
func (r *Registry) Modules() []Module {
	return r.modules
}

// Boot launches dependencies securely mapped structures activating commands evaluating explicitly internally.
func (r *Registry) Boot(parentCtx context.Context, client *bot.Client) ([]discord.ApplicationCommandCreate, []bot.EventListener, context.CancelFunc) {
	workerCtx, cancelWorkers := context.WithCancel(parentCtx)

	var commands []discord.ApplicationCommandCreate
	var listeners []bot.EventListener

	for _, m := range r.modules {
		slog.Debug("booting module", slog.String("module", m.Name()))

		m.RegisterCommands(r.router)

		if lst := m.RegisterListeners(); len(lst) > 0 {
			listeners = append(listeners, lst...)
		}

		if cmds := m.SlashCommands(); len(cmds) > 0 {
			commands = append(commands, cmds...)
		}

		for _, w := range m.Workers() {
			slog.Debug("starting worker", slog.String("module", m.Name()), slog.String("worker", w.Name))
			go w.Run(workerCtx)
		}
	}

	return commands, listeners, cancelWorkers
}
