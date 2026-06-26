// Package shared defines the cross-cutting contracts every feature module is
// built on: the dependency container, the Module plugin interface, the
// interaction Context, custom-ID encoding, error types and permission helpers.
//
// It deliberately depends only on infrastructure packages (config, cache,
// observability) and never on feature modules or the UI layer, so it can be
// imported everywhere without creating cycles.
package shared

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/cache"
	"github.com/dipu-sharma/disgo-bot/internal/config"
	"github.com/dipu-sharma/disgo-bot/internal/observability"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// Deps is the dependency-injection container constructed once in main and
// handed to every module's Init. Modules hold the dependencies they need and
// never reach for globals.
type Deps struct {
	Config  *config.Config
	Log     *zap.Logger
	DB      *bun.DB
	Cache   cache.Cache
	Session *discordgo.Session
	Metrics *observability.Metrics
}
