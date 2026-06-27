package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// ErrUnavailable is returned when the assistant isn't configured (disabled or
// missing an API key).
var ErrUnavailable = errors.New("ai assistant is not configured")

// askTimeout bounds a single completion request.
const askTimeout = 45 * time.Second

// Service holds the AI business logic: provider access, per-guild settings (with
// an in-process cache) and a cache-backed per-user rate limiter.
type Service struct {
	deps     *shared.Deps
	repo     *repo
	log      *zap.Logger
	provider Provider

	mu       sync.RWMutex
	settings map[int64]*Settings
}

// NewService constructs the AI service, wiring an Anthropic provider when the
// config supplies credentials.
func NewService(d *shared.Deps) *Service {
	s := &Service{deps: d, repo: newRepo(d.DB), log: d.Log, settings: map[int64]*Settings{}}
	if ai := d.Config.AI; ai.Ready() {
		s.provider = newAnthropicProvider(ai.APIKey, ai.Model, ai.MaxTokens, ai.BaseURL)
	}
	return s
}

// Ready reports whether the assistant can serve requests.
func (s *Service) Ready() bool { return s.provider != nil }

// Model returns the configured model ID (empty when unavailable).
func (s *Service) Model() string {
	if s.provider == nil {
		return ""
	}
	return s.provider.Model()
}

// --- settings cache ---

func (s *Service) getSettings(ctx context.Context, guildID int64) (*Settings, error) {
	s.mu.RLock()
	cached, ok := s.settings[guildID]
	s.mu.RUnlock()
	if ok {
		return cached, nil
	}
	set, err := s.repo.getSettings(ctx, guildID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.settings[guildID] = set
	s.mu.Unlock()
	return set, nil
}

func (s *Service) invalidate(guildID int64) {
	s.mu.Lock()
	delete(s.settings, guildID)
	s.mu.Unlock()
}

// Settings returns a guild's configuration (defaults when unset).
func (s *Service) Settings(ctx context.Context, guildID string) (*Settings, error) {
	return s.getSettings(ctx, pid(guildID))
}

// SetChannel sets (or clears, with 0) the opt-in assistant channel.
func (s *Service) SetChannel(ctx context.Context, guildID string, channelID int64) error {
	return s.mutate(ctx, guildID, func(set *Settings) { set.AssistantChannelID = channelID })
}

// SetSystem sets (or clears, with "") the guild's custom system prompt.
func (s *Service) SetSystem(ctx context.Context, guildID string, prompt string) error {
	return s.mutate(ctx, guildID, func(set *Settings) { set.SystemPrompt = prompt })
}

func (s *Service) mutate(ctx context.Context, guildID string, fn func(*Settings)) error {
	set, err := s.repo.getSettings(ctx, pid(guildID))
	if err != nil {
		return err
	}
	fn(set)
	if err := s.repo.saveSettings(ctx, set); err != nil {
		return err
	}
	s.invalidate(pid(guildID))
	return nil
}

// --- completion ---

// Ask sends a single-turn prompt and returns a Discord-ready reply.
func (s *Service) Ask(ctx context.Context, guildID, prompt string) (string, error) {
	if !s.Ready() {
		return "", ErrUnavailable
	}
	set, err := s.getSettings(ctx, pid(guildID))
	if err != nil {
		return "", err
	}

	cctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	answer, err := s.provider.Complete(cctx, set.system(), []Message{{Role: "user", Content: prompt}})
	if err != nil {
		return "", fmt.Errorf("ai completion: %w", err)
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return "", errors.New("the assistant returned an empty response")
	}
	return truncate(answer, maxReplyChars), nil
}

// rateLimited reports whether userID is still within their cooldown, starting a
// fresh window when they aren't. Fails open if the cache is unavailable.
func (s *Service) rateLimited(ctx context.Context, guildID, userID string) bool {
	key := fmt.Sprintf("ai:cd:%s:%s", guildID, userID)
	if ok, err := s.deps.Cache.Exists(ctx, key); err == nil && ok {
		return true
	}
	_ = s.deps.Cache.Set(ctx, key, "1", rateWindow)
	return false
}

// truncate shortens s to at most n characters (rune-safe), appending an ellipsis.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n])) + "…"
}
