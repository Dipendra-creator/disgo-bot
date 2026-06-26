package router

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// Sync registers all command definitions with Discord via a single bulk
// overwrite. When devGuildID is non-empty, commands are registered guild-scoped
// (they appear instantly, ideal for development); otherwise they register
// globally (which can take up to an hour to propagate).
func (r *Registry) Sync(s *discordgo.Session, appID, devGuildID string) error {
	if appID == "" {
		return fmt.Errorf("sync commands: empty application ID")
	}
	scope := "global"
	if devGuildID != "" {
		scope = "guild:" + devGuildID
	}

	_, err := s.ApplicationCommandBulkOverwrite(appID, devGuildID, r.defs)
	if err != nil {
		return fmt.Errorf("bulk overwrite commands (%s): %w", scope, err)
	}
	r.log.Info("synced application commands",
		zap.String("scope", scope),
		zap.Int("count", len(r.defs)),
	)
	return nil
}
