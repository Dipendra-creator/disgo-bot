package economy

import (
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// maxAmount caps a single user-supplied transfer/deposit to keep inputs sane.
const maxAmount = 1_000_000_000

func (m *Module) balanceCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "balance",
			Description: "Show your balance (or another member's)",
			Options:     []*discordgo.ApplicationCommandOption{userOpt("user", "The member to inspect", false)},
		},
		Handler: m.handleBalance,
	}
}

func (m *Module) handleBalance(c *shared.Context) error {
	if c.GuildID() == "" {
		return errGuildOnly
	}
	target := optUser(c, "user")
	if target == nil {
		target = c.User()
	}
	if target.Bot {
		return shared.UserErr("Bots don't hold a balance.")
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	a, err := m.svc.Balance(c.Ctx, c.GuildID(), target.ID)
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{balanceEmbed(set, target, a)},
	}, false)
}

func (m *Module) dailyCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "daily",
			Description: "Claim your daily reward",
		},
		Handler: m.handleDaily,
	}
}

func (m *Module) handleDaily(c *shared.Context) error {
	if c.GuildID() == "" {
		return errGuildOnly
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	earned, wallet, retryAt, err := m.svc.Daily(c.Ctx, c.GuildID(), c.User().ID)
	if errors.Is(err, ErrOnCooldown) {
		return c.Reply(&discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{cooldownEmbed("Daily already claimed", retryAt)},
		}, true)
	}
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{earnEmbed(set, "🎁 Daily reward", "You claimed your daily reward!", earned, wallet)},
	}, false)
}

func (m *Module) workCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "work",
			Description: "Work a shift to earn currency",
		},
		Handler: m.handleWork,
	}
}

// workFlavors give /work a little variety.
var workFlavors = []string{
	"You delivered packages around town.",
	"You fixed a few bugs in production.",
	"You brewed coffee for the whole office.",
	"You walked the neighbour's dogs.",
	"You streamed for a few hours.",
	"You tended the community garden.",
}

func (m *Module) handleWork(c *shared.Context) error {
	if c.GuildID() == "" {
		return errGuildOnly
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	earned, wallet, retryAt, err := m.svc.Work(c.Ctx, c.GuildID(), c.User().ID)
	if errors.Is(err, ErrOnCooldown) {
		return c.Reply(&discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{cooldownEmbed("Still resting", retryAt)},
		}, true)
	}
	if err != nil {
		return err
	}
	flavor := workFlavors[int(earned)%len(workFlavors)]
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{earnEmbed(set, "💼 Work complete", flavor, earned, wallet)},
	}, false)
}

func (m *Module) payCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "pay",
			Description: "Pay another member from your wallet",
			Options: []*discordgo.ApplicationCommandOption{
				userOpt("user", "Who to pay", true),
				intOpt("amount", "How much to send", true),
			},
		},
		Handler: m.handlePay,
	}
}

func (m *Module) handlePay(c *shared.Context) error {
	if c.GuildID() == "" {
		return errGuildOnly
	}
	target := optUser(c, "user")
	if target == nil {
		return shared.UserErr("You must specify who to pay.")
	}
	if target.Bot {
		return shared.UserErr("You can't pay a bot.")
	}
	if target.ID == c.User().ID {
		return shared.UserErr("You can't pay yourself.")
	}
	amount := optInt(c, "amount")
	if err := validAmount(amount); err != nil {
		return err
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	err = m.svc.Pay(c.Ctx, c.GuildID(), c.User().ID, target.ID, amount)
	if errors.Is(err, ErrInsufficient) {
		return shared.UserErr("You don't have enough in your wallet.")
	}
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{payEmbed(set, c.User(), target, amount)},
	}, false)
}

func (m *Module) depositCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "deposit",
			Description: "Move currency from your wallet to your bank",
			Options:     []*discordgo.ApplicationCommandOption{intOpt("amount", "How much to deposit", true)},
		},
		Handler: func(c *shared.Context) error { return m.handleMove(c, true) },
	}
}

func (m *Module) withdrawCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "withdraw",
			Description: "Move currency from your bank to your wallet",
			Options:     []*discordgo.ApplicationCommandOption{intOpt("amount", "How much to withdraw", true)},
		},
		Handler: func(c *shared.Context) error { return m.handleMove(c, false) },
	}
}

func (m *Module) handleMove(c *shared.Context, deposit bool) error {
	if c.GuildID() == "" {
		return errGuildOnly
	}
	amount := optInt(c, "amount")
	if err := validAmount(amount); err != nil {
		return err
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	a, err := m.svc.Move(c.Ctx, c.GuildID(), c.User().ID, amount, deposit)
	if errors.Is(err, ErrInsufficient) {
		where := "bank"
		if deposit {
			where = "wallet"
		}
		return shared.UserErr("You don't have that much in your %s.", where)
	}
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{moveEmbed(set, deposit, amount, a)},
	}, true)
}

// validAmount rejects non-positive or oversized amounts.
func validAmount(amount int64) error {
	if amount <= 0 {
		return shared.UserErr("Amount must be positive.")
	}
	if amount > maxAmount {
		return shared.UserErr("Amount is too large (max %d).", maxAmount)
	}
	return nil
}

// errGuildOnly is the shared guard message for guild-scoped commands.
var errGuildOnly = shared.UserErr("This command can only be used in a server.")
