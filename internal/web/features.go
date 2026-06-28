package web

import "net/http"

// features reports which management consoles the dashboard should surface for a
// guild. Each flag is true when a registered module exposes the matching
// transport-agnostic seam. The frontend builds its "Management" nav from this so
// a console never appears without a backend behind it.
type features struct {
	Moderation bool `json:"moderation"`
	Economy    bool `json:"economy"`
	Leveling   bool `json:"leveling"`
	Tickets    bool `json:"tickets"`
	Giveaways  bool `json:"giveaways"`
	AutoMod    bool `json:"automod"`
}

// handleFeatures serves GET /api/guilds/{id}/features.
func (s *Server) handleFeatures(w http.ResponseWriter, _ *http.Request, _ *Session, _ string) {
	writeJSON(w, http.StatusOK, features{
		Moderation: s.moderation != nil,
		Economy:    s.economy != nil,
		Leveling:   s.leveling != nil,
		Tickets:    s.tickets != nil,
		Giveaways:  s.giveaways != nil,
		AutoMod:    s.automod != nil,
	})
}
