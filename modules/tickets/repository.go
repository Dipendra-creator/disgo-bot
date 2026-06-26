package tickets

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
)

// ErrNoTicket is returned when a channel has no associated ticket.
var ErrNoTicket = errors.New("no ticket for channel")

type repo struct{ db *bun.DB }

func newRepo(db *bun.DB) *repo { return &repo{db: db} }

// nextNumber atomically allocates the next per-guild ticket number.
func (r *repo) nextNumber(ctx context.Context, guildID int64) (int64, error) {
	var number int64
	err := r.db.NewRaw(
		`INSERT INTO ticket_counters (guild_id, last_number)
		 VALUES (?, 1)
		 ON CONFLICT (guild_id)
		 DO UPDATE SET last_number = ticket_counters.last_number + 1
		 RETURNING last_number`, guildID).Scan(ctx, &number)
	return number, err
}

func (r *repo) insertTicket(ctx context.Context, t *Ticket) error {
	_, err := r.db.NewInsert().Model(t).Exec(ctx)
	return err
}

func (r *repo) byChannel(ctx context.Context, channelID int64) (*Ticket, error) {
	t := new(Ticket)
	err := r.db.NewSelect().Model(t).
		Where("channel_id = ?", channelID).
		Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoTicket
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// openByOpener returns the opener's current non-closed ticket, or nil.
func (r *repo) openByOpener(ctx context.Context, guildID, openerID int64) (*Ticket, error) {
	t := new(Ticket)
	err := r.db.NewSelect().Model(t).
		Where("guild_id = ? AND opener_id = ? AND status <> ?", guildID, openerID, StatusClosed).
		Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *repo) claim(ctx context.Context, channelID, claimerID int64) error {
	_, err := r.db.NewUpdate().Model((*Ticket)(nil)).
		Set("claimer_id = ?", claimerID).
		Set("status = ?", StatusClaimed).
		Where("channel_id = ?", channelID).
		Exec(ctx)
	return err
}

func (r *repo) close(ctx context.Context, channelID, closedBy int64, reason string) error {
	_, err := r.db.NewUpdate().Model((*Ticket)(nil)).
		Set("status = ?", StatusClosed).
		Set("closed_by = ?", closedBy).
		Set("close_reason = ?", reason).
		Set("closed_at = now()").
		Where("channel_id = ?", channelID).
		Exec(ctx)
	return err
}

func (r *repo) getSettings(ctx context.Context, guildID int64) (*Settings, error) {
	s := new(Settings)
	err := r.db.NewSelect().Model(s).
		Where("guild_id = ?", guildID).
		Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return &Settings{GuildID: guildID}, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// saveSettings upserts the core config (category, staff role, log channel),
// leaving panel references untouched.
func (r *repo) saveSettings(ctx context.Context, s *Settings) error {
	_, err := r.db.NewInsert().Model(s).
		On("CONFLICT (guild_id) DO UPDATE").
		Set("category_id = EXCLUDED.category_id").
		Set("staff_role_id = EXCLUDED.staff_role_id").
		Set("log_channel_id = EXCLUDED.log_channel_id").
		Set("updated_at = now()").
		Exec(ctx)
	return err
}

// setPanel records where a ticket panel was posted.
func (r *repo) setPanel(ctx context.Context, guildID, channelID, messageID int64) error {
	s := &Settings{GuildID: guildID, PanelChannelID: channelID, PanelMessageID: messageID}
	_, err := r.db.NewInsert().Model(s).
		On("CONFLICT (guild_id) DO UPDATE").
		Set("panel_channel_id = EXCLUDED.panel_channel_id").
		Set("panel_message_id = EXCLUDED.panel_message_id").
		Set("updated_at = now()").
		Exec(ctx)
	return err
}
