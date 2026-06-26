package moderation

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// highestRole returns the position of a member's highest role, or -1 when the
// member has no roles (i.e. only @everyone). A higher number outranks a lower.
func highestRole(guild *discordgo.Guild, member *discordgo.Member) int {
	if guild == nil || member == nil {
		return -1
	}
	top := -1
	for _, rid := range member.Roles {
		for _, role := range guild.Roles {
			if role.ID == rid && role.Position > top {
				top = role.Position
			}
		}
	}
	return top
}

// checkTarget enforces the identity rules that never depend on role position:
// you can't action yourself, the bot, or the guild owner.
func checkTarget(guild *discordgo.Guild, invokerID, targetID, botID string) error {
	switch {
	case targetID == invokerID:
		return shared.UserErr("You can't moderate yourself.")
	case botID != "" && targetID == botID:
		return shared.UserErr("I can't moderate myself.")
	case guild != nil && targetID == guild.OwnerID:
		return shared.UserErr("You can't moderate the server owner.")
	}
	return nil
}

// checkHierarchy enforces Discord's role hierarchy: a moderator may only action
// members strictly below them, and the bot must outrank the target to act. It
// is best-effort — when the guild or target member can't be resolved (e.g. a
// hackban on a non-member) the role checks are skipped.
func checkHierarchy(guild *discordgo.Guild, invoker, target, bot *discordgo.Member) error {
	if guild == nil || target == nil {
		return nil
	}
	if invoker != nil && invoker.User != nil && invoker.User.ID != guild.OwnerID {
		if highestRole(guild, invoker) <= highestRole(guild, target) {
			return shared.UserErr("You can't moderate someone whose highest role is above or equal to yours.")
		}
	}
	if bot != nil && highestRole(guild, bot) <= highestRole(guild, target) {
		return shared.UserErr("My highest role isn't above that member, so I can't act on them.")
	}
	return nil
}
