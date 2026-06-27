package economy

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// opTimeout bounds a single economy database operation.
const opTimeout = 8 * time.Second

// ErrOnCooldown signals that a periodic earning (daily/work) isn't available yet.
var ErrOnCooldown = errors.New("on cooldown")

// Service holds economy business logic with an in-process settings cache.
type Service struct {
	deps *shared.Deps
	repo *repo
	log  *zap.Logger

	mu    sync.RWMutex
	cache map[int64]*Settings
}

// NewService constructs the economy service.
func NewService(d *shared.Deps) *Service {
	return &Service{deps: d, repo: newRepo(d.DB), log: d.Log, cache: make(map[int64]*Settings)}
}

func (s *Service) settings(ctx context.Context, guildID int64) (*Settings, error) {
	s.mu.RLock()
	cached, ok := s.cache[guildID]
	s.mu.RUnlock()
	if ok {
		return cached, nil
	}
	set, err := s.repo.getSettings(ctx, guildID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.cache[guildID] = set
	s.mu.Unlock()
	return set, nil
}

func (s *Service) invalidate(guildID int64) {
	s.mu.Lock()
	delete(s.cache, guildID)
	s.mu.Unlock()
}

// Settings returns a guild's economy configuration (defaults when unset).
func (s *Service) Settings(ctx context.Context, guildID string) (*Settings, error) {
	return s.settings(ctx, pid(guildID))
}

// SaveSettings persists settings and refreshes the cache.
func (s *Service) SaveSettings(ctx context.Context, set *Settings) error {
	if err := s.repo.saveSettings(ctx, set); err != nil {
		return err
	}
	s.invalidate(set.GuildID)
	return nil
}

// Balance returns a member's account holdings.
func (s *Service) Balance(ctx context.Context, guildID, userID string) (*Account, error) {
	return s.repo.getAccount(ctx, pid(guildID), pid(userID))
}

// Daily claims the once-per-day reward. On cooldown it returns ErrOnCooldown
// alongside the time the next claim becomes available.
func (s *Service) Daily(ctx context.Context, guildID, userID string) (earned, wallet int64, retryAt time.Time, err error) {
	set, err := s.settings(ctx, pid(guildID))
	if err != nil {
		return 0, 0, time.Time{}, err
	}
	cutoff := time.Now().Add(-dailyCooldown)
	wallet, ok, err := s.repo.claimEarning(ctx, pid(guildID), pid(userID), set.DailyAmount, set.StartingBalance, "last_daily", cutoff)
	if err != nil {
		return 0, 0, time.Time{}, err
	}
	if !ok {
		a, gErr := s.repo.getAccount(ctx, pid(guildID), pid(userID))
		if gErr != nil {
			return 0, 0, time.Time{}, gErr
		}
		return 0, 0, a.LastDaily.Add(dailyCooldown), ErrOnCooldown
	}
	return set.DailyAmount, wallet, time.Time{}, nil
}

// Work claims a randomised work reward subject to the configured cooldown.
func (s *Service) Work(ctx context.Context, guildID, userID string) (earned, wallet int64, retryAt time.Time, err error) {
	set, err := s.settings(ctx, pid(guildID))
	if err != nil {
		return 0, 0, time.Time{}, err
	}
	cd := time.Duration(set.WorkCooldownSec) * time.Second
	cutoff := time.Now().Add(-cd)
	amount := randAmount(set.WorkMin, set.WorkMax)
	wallet, ok, err := s.repo.claimEarning(ctx, pid(guildID), pid(userID), amount, set.StartingBalance, "last_work", cutoff)
	if err != nil {
		return 0, 0, time.Time{}, err
	}
	if !ok {
		a, gErr := s.repo.getAccount(ctx, pid(guildID), pid(userID))
		if gErr != nil {
			return 0, 0, time.Time{}, gErr
		}
		return 0, 0, a.LastWork.Add(cd), ErrOnCooldown
	}
	return amount, wallet, time.Time{}, nil
}

// Pay transfers amount from one member's wallet to another's.
func (s *Service) Pay(ctx context.Context, guildID, from, to string, amount int64) error {
	set, err := s.settings(ctx, pid(guildID))
	if err != nil {
		return err
	}
	return s.repo.transfer(ctx, pid(guildID), pid(from), pid(to), amount, set.StartingBalance)
}

// Move shifts amount between a member's wallet and bank (deposit = wallet→bank).
func (s *Service) Move(ctx context.Context, guildID, userID string, amount int64, deposit bool) (*Account, error) {
	set, err := s.settings(ctx, pid(guildID))
	if err != nil {
		return nil, err
	}
	return s.repo.move(ctx, pid(guildID), pid(userID), amount, set.StartingBalance, deposit)
}

// Rich returns a page of the net-worth leaderboard plus the total ranked count.
func (s *Service) Rich(ctx context.Context, guildID string, offset, limit int) ([]Account, int, error) {
	rows, err := s.repo.richList(ctx, pid(guildID), offset, limit)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.countRich(ctx, pid(guildID))
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// --- shop ---

// Shop returns a page of shop items plus the total item count.
func (s *Service) Shop(ctx context.Context, guildID string, offset, limit int) ([]ShopItem, int, error) {
	rows, err := s.repo.listItems(ctx, pid(guildID), offset, limit)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.countItems(ctx, pid(guildID))
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// AddItem registers a purchasable shop item.
func (s *Service) AddItem(ctx context.Context, guildID, name, desc string, price int64, roleID string, stock int) error {
	it := &ShopItem{
		GuildID:     pid(guildID),
		Name:        name,
		Description: desc,
		Price:       price,
		RoleID:      pid(roleID),
		Stock:       stock,
	}
	return s.repo.addItem(ctx, it)
}

// RemoveItem deletes a shop item by name.
func (s *Service) RemoveItem(ctx context.Context, guildID, name string) (bool, error) {
	return s.repo.removeItem(ctx, pid(guildID), name)
}

// Buy purchases one unit of a shop item, granting its role when configured.
func (s *Service) Buy(ctx context.Context, guildID, userID, name string) (*ShopItem, int64, error) {
	set, err := s.settings(ctx, pid(guildID))
	if err != nil {
		return nil, 0, err
	}
	item, wallet, err := s.repo.buy(ctx, pid(guildID), pid(userID), set.StartingBalance, name)
	if err != nil {
		return nil, 0, err
	}
	if item.RoleID != 0 {
		if err := s.deps.Session.GuildMemberRoleAdd(guildID, userID, sid(item.RoleID)); err != nil {
			s.log.Warn("grant purchased role failed", zap.Error(err), zap.String("item", item.Name))
		}
	}
	return item, wallet, nil
}

// Inventory lists a member's owned items.
func (s *Service) Inventory(ctx context.Context, guildID, userID string) ([]InventoryItem, error) {
	return s.repo.inventory(ctx, pid(guildID), pid(userID))
}

// --- admin ---

// GiveBalance adds (or removes, when negative) wallet funds, clamped at zero.
func (s *Service) GiveBalance(ctx context.Context, guildID, userID string, delta int64) (int64, error) {
	set, err := s.settings(ctx, pid(guildID))
	if err != nil {
		return 0, err
	}
	return s.repo.addWallet(ctx, pid(guildID), pid(userID), delta, set.StartingBalance)
}

// SetBalance overwrites a member's wallet balance.
func (s *Service) SetBalance(ctx context.Context, guildID, userID string, wallet int64) error {
	set, err := s.settings(ctx, pid(guildID))
	if err != nil {
		return err
	}
	return s.repo.setBalance(ctx, pid(guildID), pid(userID), wallet, set.StartingBalance)
}

// ResetAccount clears a member's holdings.
func (s *Service) ResetAccount(ctx context.Context, guildID, userID string) error {
	return s.repo.resetAccount(ctx, pid(guildID), pid(userID))
}

// randAmount returns a value in [min, max]; it tolerates an inverted or equal range.
func randAmount(min, max int64) int64 {
	if max < min {
		min, max = max, min
	}
	if max == min {
		return min
	}
	return min + rand.Int63n(max-min+1)
}
