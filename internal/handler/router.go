package handler

import (
	"context"
	"strings"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

// CommandHandler handles a slash command interaction and returns any unhandled error state.
type CommandHandler func(ctx context.Context, e *events.ApplicationCommandInteractionCreate) error

// ButtonHandler handles a component interaction of type button.
type ButtonHandler func(ctx context.Context, e *events.ComponentInteractionCreate) error

// ModalHandler handles a modal submit interaction.
type ModalHandler func(ctx context.Context, e *events.ModalSubmitInteractionCreate) error

// SelectMenuHandler handles a select menu interaction.
type SelectMenuHandler func(ctx context.Context, e *events.ComponentInteractionCreate) error

// Router dispatches interactions to configured registered handlers.
// The data mappings represent O(1) direct resolutions logic flows avoiding repetitive checks.
type Router struct {
	commands    map[string]CommandHandler
	subGroups   map[string]map[string]map[string]CommandHandler
	subCommands map[string]map[string]CommandHandler
	buttons     map[string]ButtonHandler
	modals      map[string]ModalHandler
	selects     map[string]SelectMenuHandler
	middleware  []Middleware
}

// NewRouter constructs context structure enabling O(1) map resolution algorithms on interactions.
func NewRouter() *Router {
	return &Router{
		commands:    make(map[string]CommandHandler),
		subGroups:   make(map[string]map[string]map[string]CommandHandler),
		subCommands: make(map[string]map[string]CommandHandler),
		buttons:     make(map[string]ButtonHandler),
		modals:      make(map[string]ModalHandler),
		selects:     make(map[string]SelectMenuHandler),
	}
}

// Command registers a top-level slash command discrete mapping handler.
func (r *Router) Command(name string, h CommandHandler) {
	r.commands[name] = h
}

// SubCommand registers a subcommand context mapped securely logically.
func (r *Router) SubCommand(parent, sub string, h CommandHandler) {
	if _, ok := r.subCommands[parent]; !ok {
		r.subCommands[parent] = make(map[string]CommandHandler)
	}
	r.subCommands[parent][sub] = h
}

// SubCommandGroup registers a deeply nested group layout sub-action.
func (r *Router) SubCommandGroup(parent, group, sub string, h CommandHandler) {
	if _, ok := r.subGroups[parent]; !ok {
		r.subGroups[parent] = make(map[string]map[string]CommandHandler)
	}
	if _, ok := r.subGroups[parent][group]; !ok {
		r.subGroups[parent][group] = make(map[string]CommandHandler)
	}
	r.subGroups[parent][group][sub] = h
}

// Button binds interactions matching prefix identifier rules strictly.
func (r *Router) Button(prefix string, h ButtonHandler) {
	r.buttons[prefix] = h
}

// Select binds interactions matching prefix identifier rules strictly for select menus.
func (r *Router) Select(prefix string, h SelectMenuHandler) {
	r.selects[prefix] = h
}

// Use defines the standard middleware operational stack mapped internally.
func (r *Router) Use(mw ...Middleware) {
	r.middleware = append(r.middleware, mw...)
}

// CommandListener synthesizes standard interactions into localized evaluation execution patterns implicitly.
func (r *Router) CommandListener() bot.EventListener {
	return bot.NewListenerFunc(func(e *events.ApplicationCommandInteractionCreate) {
		data := e.SlashCommandInteractionData()
		name := data.CommandName()
		var handler CommandHandler

		if data.SubCommandGroupName != nil {
			if sg, ok := r.subGroups[name]; ok {
				if g, ok := sg[*data.SubCommandGroupName]; ok {
					handler = g[*data.SubCommandName]
				}
			}
		} else if data.SubCommandName != nil {
			if sc, ok := r.subCommands[name]; ok {
				handler = sc[*data.SubCommandName]
			}
		} else {
			handler = r.commands[name]
		}

		if handler == nil {
			return
		}

		// Implement chain wrapping execution natively top-down implicitly mapping context dependencies.
		h := handler
		for i := len(r.middleware) - 1; i >= 0; i-- {
			h = r.middleware[i](h)
		}

		_ = h(context.Background(), e)
	})
}

// ButtonListener evaluates custom id prefix bindings natively executing context responses.
func (r *Router) ButtonListener() bot.EventListener {
	return bot.NewListenerFunc(func(e *events.ComponentInteractionCreate) {
		if e.Data.Type() != discord.ComponentTypeButton {
			return
		}

		customID := e.Data.CustomID()

		var handler ButtonHandler
		var longest string

		for prefix, h := range r.buttons {
			if strings.HasPrefix(customID, prefix) && len(prefix) > len(longest) {
				longest = prefix
				handler = h
			}
		}

		if handler == nil {
			return
		}

		// Safe fallback standard contextual response mapping evaluation patterns implicitly structured contextually
		_ = handler(context.Background(), e)
	})
}

// SelectListener evaluates custom id prefix bindings natively executing context responses for select menus.
func (r *Router) SelectListener() bot.EventListener {
	return bot.NewListenerFunc(func(e *events.ComponentInteractionCreate) {
		if e.Data.Type() == discord.ComponentTypeButton {
			return
		}

		customID := e.Data.CustomID()

		var handler SelectMenuHandler
		var longest string

		for prefix, h := range r.selects {
			if strings.HasPrefix(customID, prefix) && len(prefix) > len(longest) {
				longest = prefix
				handler = h
			}
		}

		if handler == nil {
			return
		}

		_ = handler(context.Background(), e)
	})
}

// Modal registers a modal submit discrete mapping handler.
func (r *Router) Modal(prefix string, h ModalHandler) {
	r.modals[prefix] = h
}

// ModalListener evaluates custom id prefix bindings natively executing context responses for modal submits.
func (r *Router) ModalListener() bot.EventListener {
	return bot.NewListenerFunc(func(e *events.ModalSubmitInteractionCreate) {
		customID := e.Data.CustomID

		var handler ModalHandler
		var longest string

		for prefix, h := range r.modals {
			if strings.HasPrefix(customID, prefix) && len(prefix) > len(longest) {
				longest = prefix
				handler = h
			}
		}

		if handler == nil {
			return
		}

		_ = handler(context.Background(), e)
	})
}
