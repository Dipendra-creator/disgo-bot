package leveling

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
)

// usersPerPage bounds how many ranks render on a single leaderboard page.
const usersPerPage = 10

// levelUpEmbed congratulates a member on reaching a new level.
func levelUpEmbed(u *discordgo.User, level int) *discordgo.MessageEmbed {
	return ui.NewEmbed().
		Color(ui.ColorBrand).
		Title(ui.EmojiSparkles + " Level up!").
		Description(fmt.Sprintf("%s reached **level %d**.", u.Mention(), level)).
		Thumbnail(u.AvatarURL("128")).
		Timestamp().Build()
}

// rankEmbed renders a member's rank card.
func rankEmbed(u *discordgo.User, ul *UserLevel, p progress, rank, totalRanked int) *discordgo.MessageEmbed {
	e := ui.NewEmbed().
		Color(ui.ColorInfo).
		Author("Rank • "+u.String(), u.AvatarURL("128")).
		Thumbnail(u.AvatarURL("256"))

	if ul.XP == 0 {
		return e.Description(u.Mention() + " hasn't earned any XP yet. Start chatting!").Build()
	}

	rankStr := "—"
	if rank > 0 {
		rankStr = fmt.Sprintf("#%d / %s", rank, humanize.Comma(totalRanked))
	}
	e.Field("Level", fmt.Sprintf("%d", p.Level), true)
	e.Field("Rank", rankStr, true)
	e.Field("Total XP", humanize.Comma(int(p.Total)), true)
	e.Field(
		fmt.Sprintf("Progress to level %d", p.Level+1),
		fmt.Sprintf("%s\n%s / %s XP", ui.ProgressBar(int(p.Into), int(p.Need), 16), humanize.Comma(int(p.Into)), humanize.Comma(int(p.Need))),
		false,
	)
	return e.Footer("disgo • leveling", "").Timestamp().Build()
}

// leaderboardEmbed renders one page of the XP leaderboard.
func leaderboardEmbed(guildName string, rows []UserLevel, page, totalPages, totalRanked, offset int) *discordgo.MessageEmbed {
	e := ui.NewEmbed().
		Color(ui.ColorBrand).
		Title("🏆 " + guildName + " — Leaderboard")

	if len(rows) == 0 {
		return e.Description("No ranked members yet. Start chatting to earn XP!").Build()
	}

	medals := map[int]string{0: "🥇", 1: "🥈", 2: "🥉"}
	body := ""
	for i, r := range rows {
		pos := offset + i
		badge := medals[pos]
		if badge == "" {
			badge = fmt.Sprintf("`#%d`", pos+1)
		}
		body += fmt.Sprintf("%s <@%s> — **Level %d** · %s XP\n", badge, sid(r.UserID), r.Level, humanize.Comma(int(r.XP)))
	}
	e.Description(body)
	e.Footer(fmt.Sprintf("disgo • leveling · page %d/%d · %s ranked", page+1, totalPages, humanize.Comma(totalRanked)), "")
	return e.Build()
}

// rewardsEmbed lists configured level-reward roles.
func rewardsEmbed(rewards []Reward) *discordgo.MessageEmbed {
	e := ui.NewEmbed().Color(ui.ColorInfo).Title("🎁 Level rewards")
	if len(rewards) == 0 {
		return e.Description("No level rewards configured. Add one with `/level-role add`.").Build()
	}
	body := ""
	for _, rw := range rewards {
		body += fmt.Sprintf("**Level %d** → <@&%s>\n", rw.Level, sid(rw.RoleID))
	}
	return e.Description(body).Footer("disgo • leveling", "").Build()
}

// settingsEmbed summarises the leveling configuration.
func settingsEmbed(s *Settings) *discordgo.MessageEmbed {
	state := "✅ Enabled"
	if !s.Enabled {
		state = "❌ Disabled"
	}
	announce := "in the active channel"
	if !s.AnnounceEnabled {
		announce = "off"
	} else if s.AnnounceChannelID != 0 {
		announce = fmt.Sprintf("<#%s>", sid(s.AnnounceChannelID))
	}
	stack := "highest only"
	if s.StackRoles {
		stack = "stacked"
	}
	return ui.NewEmbed().
		Color(ui.ColorInfo).
		Title("⚙️ Leveling configuration").
		Field("Status", state, true).
		Field("XP per message", fmt.Sprintf("%d–%d", s.XPMin, s.XPMax), true).
		Field("Cooldown", fmt.Sprintf("%ds", s.XPCooldownSeconds), true).
		Field("Level-up announcements", announce, true).
		Field("Reward roles", stack, true).
		Footer("disgo • leveling", "").Build()
}
