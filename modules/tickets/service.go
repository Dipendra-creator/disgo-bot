package tickets

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// ticketMemberPerms is the permission set granted to a ticket's participants
// (opener and staff) on the private channel.
const ticketMemberPerms = discordgo.PermissionViewChannel |
	discordgo.PermissionSendMessages |
	discordgo.PermissionReadMessageHistory |
	discordgo.PermissionAttachFiles |
	discordgo.PermissionEmbedLinks

// transcriptScan caps how many recent messages are captured into a transcript.
const transcriptScan = 100

// Service holds the ticket business logic, independent of the interaction layer.
type Service struct {
	deps *shared.Deps
	repo *repo
	log  *zap.Logger
}

// NewService constructs the ticket service.
func NewService(d *shared.Deps) *Service {
	return &Service{deps: d, repo: newRepo(d.DB), log: d.Log}
}

func (s *Service) botID() string {
	if s.deps.Session.State != nil && s.deps.Session.State.User != nil {
		return s.deps.Session.State.User.ID
	}
	return ""
}

// Setup persists the core ticket configuration for a guild.
func (s *Service) Setup(ctx context.Context, guildID string, categoryID, staffRoleID, logChannelID int64) error {
	return s.repo.saveSettings(ctx, &Settings{
		GuildID:      pid(guildID),
		CategoryID:   categoryID,
		StaffRoleID:  staffRoleID,
		LogChannelID: logChannelID,
	})
}

// Settings returns a guild's ticket configuration (defaults when unset).
func (s *Service) Settings(ctx context.Context, guildID string) (*Settings, error) {
	return s.repo.getSettings(ctx, pid(guildID))
}

// TicketByChannel resolves the ticket a channel belongs to.
func (s *Service) TicketByChannel(ctx context.Context, channelID string) (*Ticket, error) {
	return s.repo.byChannel(ctx, pid(channelID))
}

// PostPanel publishes a ticket panel (a Components-v2 message with an Open
// button) to a channel and records its location.
func (s *Service) PostPanel(ctx context.Context, guildID, channelID, title, desc string) error {
	msg, err := s.deps.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Flags:      discordgo.MessageFlagsIsComponentsV2,
		Components: panelComponents(title, desc),
	})
	if err != nil {
		return fmt.Errorf("post panel: %w", err)
	}
	if err := s.repo.setPanel(ctx, pid(guildID), pid(channelID), pid(msg.ID)); err != nil {
		s.log.Warn("record panel failed", zap.Error(err))
	}
	return nil
}

// OpenTicket creates a private ticket channel for the opener, posts the welcome
// message and records the ticket. It enforces one open ticket per user.
func (s *Service) OpenTicket(ctx context.Context, guildID string, opener *discordgo.User, subject string) (*Ticket, error) {
	set, err := s.repo.getSettings(ctx, pid(guildID))
	if err != nil {
		return nil, err
	}
	if !set.Configured() {
		return nil, shared.UserErr("Tickets aren't set up yet. An admin needs to run `/ticket-setup`.")
	}
	if existing, _ := s.repo.openByOpener(ctx, pid(guildID), pid(opener.ID)); existing != nil {
		return nil, shared.UserErr("You already have an open ticket: <#%s>.", sid(existing.ChannelID))
	}

	number, err := s.repo.nextNumber(ctx, pid(guildID))
	if err != nil {
		return nil, err
	}

	overwrites := buildOverwrites(guildID, opener.ID, s.botID(), set.StaffRoleID)
	ch, err := s.deps.Session.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:                 fmt.Sprintf("ticket-%04d", number),
		Type:                 discordgo.ChannelTypeGuildText,
		Topic:                fmt.Sprintf("Ticket #%d • opened by %s (%s)", number, opener.String(), opener.ID),
		ParentID:             sid(set.CategoryID),
		PermissionOverwrites: overwrites,
	})
	if err != nil {
		return nil, fmt.Errorf("create ticket channel: %w", err)
	}

	t := &Ticket{
		GuildID:   pid(guildID),
		Number:    number,
		ChannelID: pid(ch.ID),
		OpenerID:  pid(opener.ID),
		Subject:   subject,
		Status:    StatusOpen,
	}
	if err := s.repo.insertTicket(ctx, t); err != nil {
		// Roll back the orphaned channel so a DB failure leaves no mess.
		if _, derr := s.deps.Session.ChannelDelete(ch.ID); derr != nil {
			s.log.Warn("rollback ticket channel failed", zap.Error(derr), zap.String("channel", ch.ID))
		}
		return nil, err
	}

	s.sendWelcome(ch.ID, set, t, opener)
	return t, nil
}

func (s *Service) sendWelcome(channelID string, set *Settings, t *Ticket, opener *discordgo.User) {
	content := opener.Mention()
	if set.StaffRoleID != 0 {
		content += " <@&" + sid(set.StaffRoleID) + ">"
	}
	_, err := s.deps.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:    content,
		Embeds:     []*discordgo.MessageEmbed{welcomeEmbed(t, opener)},
		Components: []discordgo.MessageComponent{ticketControls()},
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers, discordgo.AllowedMentionTypeRoles},
		},
	})
	if err != nil {
		s.log.Warn("send ticket welcome failed", zap.Error(err), zap.String("channel", channelID))
	}
}

// ClaimTicket assigns the ticket to a staff member.
func (s *Service) ClaimTicket(ctx context.Context, channelID string, claimer *discordgo.User) (*Ticket, error) {
	t, err := s.repo.byChannel(ctx, pid(channelID))
	if err != nil {
		return nil, err
	}
	if t.Status == StatusClosed {
		return nil, shared.UserErr("This ticket is closed.")
	}
	if t.ClaimerID != 0 {
		return nil, shared.UserErr("This ticket is already claimed by <@%s>.", sid(t.ClaimerID))
	}
	if err := s.repo.claim(ctx, pid(channelID), pid(claimer.ID)); err != nil {
		return nil, err
	}
	t.ClaimerID = pid(claimer.ID)
	t.Status = StatusClaimed
	return t, nil
}

// CloseTicket posts a transcript to the log channel (if configured), marks the
// ticket closed and deletes the channel.
func (s *Service) CloseTicket(ctx context.Context, guildID, channelID string, closer *discordgo.User, reason string) (*Ticket, error) {
	t, err := s.repo.byChannel(ctx, pid(channelID))
	if err != nil {
		return nil, err
	}
	if t.Status == StatusClosed {
		return nil, shared.UserErr("This ticket is already closed.")
	}

	if set, _ := s.repo.getSettings(ctx, pid(guildID)); set != nil && set.LogChannelID != 0 {
		s.postTranscript(channelID, set.LogChannelID, t, closer, reason)
	}
	if err := s.repo.close(ctx, pid(channelID), pid(closer.ID), reason); err != nil {
		return nil, err
	}
	t.Status = StatusClosed
	t.ClosedBy = pid(closer.ID)
	t.CloseReason = reason

	if _, err := s.deps.Session.ChannelDelete(channelID); err != nil {
		s.log.Warn("delete ticket channel failed", zap.Error(err), zap.String("channel", channelID))
	}
	return t, nil
}

func (s *Service) postTranscript(channelID string, logChannelID int64, t *Ticket, closer *discordgo.User, reason string) {
	msgs, err := s.deps.Session.ChannelMessages(channelID, transcriptScan, "", "", "")
	if err != nil {
		s.log.Warn("fetch transcript failed", zap.Error(err))
	}

	var b strings.Builder
	// ChannelMessages returns newest-first; emit chronologically.
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		author := "unknown"
		if m.Author != nil {
			author = m.Author.String()
		}
		line := m.Content
		if line == "" && len(m.Embeds) > 0 {
			line = "[embed]"
		}
		if len(m.Attachments) > 0 {
			line += " [attachment]"
		}
		b.WriteString(fmt.Sprintf("[%s] %s: %s\n", m.Timestamp.UTC().Format("2006-01-02 15:04:05"), author, line))
	}
	body := b.String()
	if body == "" {
		body = "(no messages)\n"
	}

	send := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{transcriptEmbed(t, closer, reason, len(msgs))},
		Files: []*discordgo.File{{
			Name:        fmt.Sprintf("transcript-%04d.txt", t.Number),
			ContentType: "text/plain",
			Reader:      strings.NewReader(body),
		}},
	}
	if _, err := s.deps.Session.ChannelMessageSendComplex(sid(logChannelID), send); err != nil {
		s.log.Warn("post transcript failed", zap.Error(err), zap.Int64("channel", logChannelID))
	}
}

// AddUser grants a user access to a ticket channel.
func (s *Service) AddUser(channelID, userID string) error {
	return s.deps.Session.ChannelPermissionSet(channelID, userID, discordgo.PermissionOverwriteTypeMember, ticketMemberPerms, 0)
}

// RemoveUser revokes a user's access to a ticket channel.
func (s *Service) RemoveUser(channelID, userID string) error {
	return s.deps.Session.ChannelPermissionSet(channelID, userID, discordgo.PermissionOverwriteTypeMember, 0, discordgo.PermissionViewChannel)
}

// buildOverwrites computes the permission overwrites for a new ticket channel:
// hidden from @everyone, visible to the opener, the bot, and the staff role.
func buildOverwrites(guildID, openerID, botID string, staffRoleID int64) []*discordgo.PermissionOverwrite {
	ow := []*discordgo.PermissionOverwrite{
		{ID: guildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: discordgo.PermissionViewChannel},
		{ID: openerID, Type: discordgo.PermissionOverwriteTypeMember, Allow: ticketMemberPerms},
	}
	if botID != "" {
		ow = append(ow, &discordgo.PermissionOverwrite{
			ID:    botID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: ticketMemberPerms | discordgo.PermissionManageChannels,
		})
	}
	if staffRoleID != 0 {
		ow = append(ow, &discordgo.PermissionOverwrite{
			ID:    sid(staffRoleID),
			Type:  discordgo.PermissionOverwriteTypeRole,
			Allow: ticketMemberPerms,
		})
	}
	return ow
}
