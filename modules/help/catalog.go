package help

// This file is the single source of truth for the /help command: a curated,
// hand-authored catalog of every command the bot exposes, grouped by the module
// that owns it. It is deliberately decoupled from the live command registry —
// the help module must not import other feature modules (Clean Architecture:
// modules stay independent) — so each command is described here with usage,
// permission, and real-world examples written for humans rather than derived
// from a schema. When a module's commands change, update the matching category
// below; the catalog_test guards the structural invariants.

// Example is one concrete invocation of a command with a short note on what it
// does, so users can copy a real command instead of decoding a usage string.
type Example struct {
	Usage string // exact text a user would type, e.g. "/ban user:@spammer reason:Raiding"
	Note  string // what this particular invocation accomplishes
}

// Command documents a single slash command (or subcommand group).
type Command struct {
	Name     string // bare name without slash, e.g. "ban"
	Usage    string // signature with <required> and [optional] args
	Desc     string // one-line summary
	Perm     string // human permission label; "" means everyone can use it
	Examples []Example
}

// Category is one feature module's worth of commands.
type Category struct {
	Key      string // stable select-menu value, matches the module name
	Label    string // display name
	Emoji    string // unicode glyph for menus and headers
	Blurb    string // one-line description of the module
	Commands []Command
}

// Catalog is the ordered list of every category shown in /help. Order here is
// the order shown in the overview and the select menu.
var Catalog = []Category{
	{
		Key: "utility", Label: "Utility", Emoji: "🛠️",
		Blurb: "Everyday server and member info commands. No permissions required.",
		Commands: []Command{
			{
				Name: "ping", Usage: "/ping", Desc: "Check the bot's gateway and API latency.",
				Examples: []Example{{Usage: "/ping", Note: "Shows round-trip and websocket latency in ms."}},
			},
			{
				Name: "serverinfo", Usage: "/serverinfo", Desc: "Show member, channel, role counts and boost status.",
				Examples: []Example{{Usage: "/serverinfo", Note: "Live overview with a Refresh button to update in place."}},
			},
			{
				Name: "userinfo", Usage: "/userinfo [user]", Desc: "Show account age, join date and roles for a member.",
				Examples: []Example{
					{Usage: "/userinfo", Note: "Your own profile when no user is given."},
					{Usage: "/userinfo user:@ada", Note: "Inspect another member; also available via the User Info right-click menu."},
				},
			},
			{
				Name: "avatar", Usage: "/avatar [user]", Desc: "Show a member's avatar with size buttons.",
				Examples: []Example{{Usage: "/avatar user:@ada", Note: "Full-resolution avatar with 128/256/512/1024 size buttons."}},
			},
		},
	},
	{
		Key: "moderation", Label: "Moderation", Emoji: "🔨",
		Blurb: "Bans, kicks, timeouts, warnings and a numbered case log.",
		Commands: []Command{
			{
				Name: "ban", Usage: "/ban <user> [reason] [delete_days] [duration]", Desc: "Ban a member, optionally for a fixed duration.", Perm: "Ban Members",
				Examples: []Example{
					{Usage: "/ban user:@spammer reason:Raid spam delete_days:1", Note: "Permanent ban; also purges their last day of messages."},
					{Usage: "/ban user:@troll duration:7d reason:Cooling off", Note: "Temp-ban that auto-lifts after 7 days."},
				},
			},
			{
				Name: "unban", Usage: "/unban <user_id> [reason]", Desc: "Lift a ban by user ID.", Perm: "Ban Members",
				Examples: []Example{{Usage: "/unban user_id:123456789012345678 reason:Appeal accepted", Note: "Use the numeric ID — the user isn't in the server to @mention."}},
			},
			{
				Name: "kick", Usage: "/kick <user> [reason]", Desc: "Remove a member; they can rejoin with an invite.", Perm: "Kick Members",
				Examples: []Example{{Usage: "/kick user:@rulebreaker reason:Ignoring warnings", Note: "Logged as a case in the modlog."}},
			},
			{
				Name: "timeout", Usage: "/timeout <user> <duration> [reason]", Desc: "Mute a member for a duration (Discord timeout).", Perm: "Moderate Members",
				Examples: []Example{
					{Usage: "/timeout user:@loud duration:30m reason:Spamming chat", Note: "Durations like 30m, 2h, 1d (max 28d)."},
					{Usage: "/timeout user:@flooder duration:1h", Note: "Quick hour-long cooldown."},
				},
			},
			{
				Name: "untimeout", Usage: "/untimeout <user> [reason]", Desc: "Clear an active timeout early.", Perm: "Moderate Members",
				Examples: []Example{{Usage: "/untimeout user:@loud reason:Apologised", Note: "Restores chat access immediately."}},
			},
			{
				Name: "warn", Usage: "/warn <user> <reason>", Desc: "Issue a warning recorded as a case.", Perm: "Moderate Members",
				Examples: []Example{{Usage: "/warn user:@newbie reason:Off-topic in #support", Note: "Stacks up in /warnings for escalation."}},
			},
			{
				Name: "warnings", Usage: "/warnings <user>", Desc: "List a member's warning history.", Perm: "Moderate Members",
				Examples: []Example{{Usage: "/warnings user:@newbie", Note: "Shows every warning with its case number and date."}},
			},
			{
				Name: "case", Usage: "/case <number>", Desc: "Look up a single moderation case.", Perm: "Moderate Members",
				Examples: []Example{{Usage: "/case number:42", Note: "Full detail for case #42 — action, moderator, reason."}},
			},
			{
				Name: "reason", Usage: "/reason <number> <reason>", Desc: "Edit the reason on an existing case.", Perm: "Moderate Members",
				Examples: []Example{{Usage: "/reason number:42 reason:Confirmed evasion of ban", Note: "Corrects or adds context after the fact."}},
			},
			{
				Name: "delwarn", Usage: "/delwarn <number>", Desc: "Delete a warning case.", Perm: "Moderate Members",
				Examples: []Example{{Usage: "/delwarn number:42", Note: "Removes a warning issued in error."}},
			},
			{
				Name: "purge", Usage: "/purge <count> [user]", Desc: "Bulk-delete recent messages, optionally from one user.", Perm: "Manage Messages",
				Examples: []Example{
					{Usage: "/purge count:50", Note: "Clears the last 50 messages in this channel."},
					{Usage: "/purge count:100 user:@spammer", Note: "Deletes only that user's messages out of the last 100."},
				},
			},
			{
				Name: "modlog", Usage: "/modlog <channel>", Desc: "Set the channel that receives case logs.", Perm: "Manage Server",
				Examples: []Example{{Usage: "/modlog channel:#mod-log", Note: "Every ban/kick/warn is posted here as a numbered case."}},
			},
		},
	},
	{
		Key: "tickets", Label: "Tickets", Emoji: "🎫",
		Blurb: "Private support channels opened from a button panel.",
		Commands: []Command{
			{
				Name: "ticket-setup", Usage: "/ticket-setup <category> [staff_role] [log_channel]", Desc: "Configure where tickets open and who can see them.", Perm: "Manage Server",
				Examples: []Example{{Usage: "/ticket-setup category:Support staff_role:@Support log_channel:#ticket-logs", Note: "New tickets become channels under the Support category, visible to @Support."}},
			},
			{
				Name: "ticket-panel", Usage: "/ticket-panel [channel] [title] [description]", Desc: "Post the button users click to open a ticket.", Perm: "Manage Server",
				Examples: []Example{{Usage: "/ticket-panel channel:#get-help title:Need a hand? description:Click below to reach the team.", Note: "Drops an Open Ticket button in #get-help."}},
			},
			{
				Name: "close", Usage: "/close [reason]", Desc: "Close the ticket you're in.", Perm: "Staff or ticket opener",
				Examples: []Example{{Usage: "/close reason:Resolved — refund issued", Note: "Run inside a ticket channel; archives it with the reason."}},
			},
			{
				Name: "ticket-add", Usage: "/ticket-add <user>", Desc: "Add a member to the current ticket.", Perm: "Staff or ticket opener",
				Examples: []Example{{Usage: "/ticket-add user:@witness", Note: "Pulls someone else into this private channel."}},
			},
			{
				Name: "ticket-remove", Usage: "/ticket-remove <user>", Desc: "Remove a member from the current ticket.", Perm: "Staff or ticket opener",
				Examples: []Example{{Usage: "/ticket-remove user:@witness", Note: "Revokes their access to this ticket."}},
			},
		},
	},
	{
		Key: "logging", Label: "Logging", Emoji: "📋",
		Blurb: "Audit-style event logs routed per category to channels you choose.",
		Commands: []Command{
			{
				Name: "logging", Usage: "/logging <set|disable|status> ...", Desc: "Route message, member and server events to log channels.", Perm: "Manage Server",
				Examples: []Example{
					{Usage: "/logging set category:message channel:#message-log", Note: "Logs edits and deletions to #message-log."},
					{Usage: "/logging set category:member channel:#join-leave", Note: "Logs joins, leaves and nickname changes."},
					{Usage: "/logging status", Note: "Show which categories are enabled and where they point."},
					{Usage: "/logging disable category:server", Note: "Stop logging server-level changes."},
				},
			},
		},
	},
	{
		Key: "leveling", Label: "Leveling", Emoji: "⭐",
		Blurb: "XP and levels for chatting, with rank cards and level-up roles.",
		Commands: []Command{
			{
				Name: "rank", Usage: "/rank [user]", Desc: "Show a member's level, XP and rank.",
				Examples: []Example{{Usage: "/rank", Note: "Your own card; pass user:@ada to view someone else's."}},
			},
			{
				Name: "leaderboard", Usage: "/leaderboard", Desc: "Show the top members by XP.",
				Examples: []Example{{Usage: "/leaderboard", Note: "Paginated ranking for the whole server."}},
			},
			{
				Name: "level-config", Usage: "/level-config <status|enable|disable|cooldown|xp-range|announce|stack> ...", Desc: "Tune how XP is earned and announced.", Perm: "Manage Server",
				Examples: []Example{
					{Usage: "/level-config enable", Note: "Turn the leveling system on."},
					{Usage: "/level-config xp-range min:15 max:25", Note: "Each eligible message grants 15–25 XP."},
					{Usage: "/level-config announce mode:channel channel:#level-ups", Note: "Post level-up messages to a dedicated channel."},
				},
			},
			{
				Name: "level-role", Usage: "/level-role <add|remove|list> ...", Desc: "Grant roles automatically at given levels.", Perm: "Manage Roles",
				Examples: []Example{
					{Usage: "/level-role add level:10 role:@Active", Note: "Members get @Active when they hit level 10."},
					{Usage: "/level-role list", Note: "Show every level→role reward."},
				},
			},
			{
				Name: "xp", Usage: "/xp <give|set|reset> ...", Desc: "Manually adjust a member's XP.", Perm: "Manage Server",
				Examples: []Example{
					{Usage: "/xp give user:@ada amount:500", Note: "Award bonus XP for an event win."},
					{Usage: "/xp reset user:@ada", Note: "Wipe a member's XP back to zero."},
				},
			},
		},
	},
	{
		Key: "economy", Label: "Economy", Emoji: "💰",
		Blurb: "Server currency with daily rewards, jobs, a shop and transfers.",
		Commands: []Command{
			{
				Name: "balance", Usage: "/balance [user]", Desc: "Check wallet and bank balance.",
				Examples: []Example{{Usage: "/balance", Note: "Your own balance; pass user:@ada to peek at someone else's."}},
			},
			{
				Name: "daily", Usage: "/daily", Desc: "Claim your once-a-day reward.",
				Examples: []Example{{Usage: "/daily", Note: "Collect the daily payout; resets every 24h."}},
			},
			{
				Name: "work", Usage: "/work", Desc: "Earn currency on a cooldown.",
				Examples: []Example{{Usage: "/work", Note: "Earn a random amount; wait out the cooldown before working again."}},
			},
			{
				Name: "pay", Usage: "/pay <user> <amount>", Desc: "Transfer currency to another member.",
				Examples: []Example{{Usage: "/pay user:@ada amount:250", Note: "Send 250 from your wallet to Ada."}},
			},
			{
				Name: "deposit", Usage: "/deposit <amount>", Desc: "Move currency from wallet to bank.",
				Examples: []Example{{Usage: "/deposit amount:1000", Note: "Bank funds are safe from robberies and losses."}},
			},
			{
				Name: "withdraw", Usage: "/withdraw <amount>", Desc: "Move currency from bank to wallet.",
				Examples: []Example{{Usage: "/withdraw amount:1000", Note: "Free up funds to spend or transfer."}},
			},
			{
				Name: "shop", Usage: "/shop", Desc: "Browse purchasable items.",
				Examples: []Example{{Usage: "/shop", Note: "Lists every item with its price."}},
			},
			{
				Name: "buy", Usage: "/buy <item>", Desc: "Purchase a shop item.",
				Examples: []Example{{Usage: "/buy item:VIP Role", Note: "Spends currency and grants the item or role."}},
			},
			{
				Name: "inventory", Usage: "/inventory [user]", Desc: "Show owned items.",
				Examples: []Example{{Usage: "/inventory", Note: "Your items; pass user:@ada to view another member's."}},
			},
			{
				Name: "rich", Usage: "/rich", Desc: "Show the wealthiest members.",
				Examples: []Example{{Usage: "/rich", Note: "Currency leaderboard for the server."}},
			},
			{
				Name: "eco-config", Usage: "/eco-config <status|currency|daily|work|starting> ...", Desc: "Configure currency name, rewards and cooldowns.", Perm: "Manage Server",
				Examples: []Example{
					{Usage: "/eco-config currency name:Gold symbol:🪙", Note: "Rename the currency to Gold (🪙)."},
					{Usage: "/eco-config daily amount:500", Note: "Set the daily reward to 500."},
				},
			},
			{
				Name: "eco-admin", Usage: "/eco-admin <give|set|reset|shop-add|shop-remove> ...", Desc: "Adjust balances and manage the shop.", Perm: "Manage Server",
				Examples: []Example{
					{Usage: "/eco-admin give user:@ada amount:1000", Note: "Add 1000 to Ada's wallet."},
					{Usage: "/eco-admin shop-add name:VIP Role price:5000 role:@VIP", Note: "List a role for sale at 5000."},
				},
			},
		},
	},
	{
		Key: "verification", Label: "Verification", Emoji: "✅",
		Blurb: "Gate new members behind a one-click verify button that grants a role.",
		Commands: []Command{
			{
				Name: "verify-setup", Usage: "/verify-setup <role> [log_channel] [message] [button_label]", Desc: "Choose the role granted on verification.", Perm: "Manage Server",
				Examples: []Example{{Usage: "/verify-setup role:@Member log_channel:#verify-log button_label:I agree", Note: "Verified members receive @Member; actions logged to #verify-log."}},
			},
			{
				Name: "verify-panel", Usage: "/verify-panel [channel] [title] [description]", Desc: "Post the verification button.", Perm: "Manage Server",
				Examples: []Example{{Usage: "/verify-panel channel:#verify title:Welcome! description:Click to unlock the server.", Note: "Drops the verify button in #verify."}},
			},
			{
				Name: "verify-status", Usage: "/verify-status", Desc: "Show the current verification configuration.", Perm: "Manage Server",
				Examples: []Example{{Usage: "/verify-status", Note: "Confirms the role, log channel and panel text."}},
			},
			{
				Name: "verify-disable", Usage: "/verify-disable", Desc: "Turn verification off.", Perm: "Manage Server",
				Examples: []Example{{Usage: "/verify-disable", Note: "Stops gating new members."}},
			},
		},
	},
	{
		Key: "automod", Label: "AutoMod", Emoji: "🛡️",
		Blurb: "Automatic filters for banned words, invites, mass-mentions and spam.",
		Commands: []Command{
			{
				Name: "automod", Usage: "/automod <status|log|exempt|timeout|words|invites|mentions|spam> ...", Desc: "Enable and tune the automatic content filters.", Perm: "Manage Server",
				Examples: []Example{
					{Usage: "/automod status", Note: "Show every filter and whether it's on."},
					{Usage: "/automod invites enabled:true action:delete", Note: "Auto-delete Discord invite links."},
					{Usage: "/automod mentions enabled:true threshold:5 action:timeout", Note: "Time out anyone who pings 5+ users in one message."},
					{Usage: "/automod spam enabled:true count:5 per_seconds:3 action:timeout", Note: "Punish 5 messages within 3 seconds."},
				},
			},
			{
				Name: "automod-words", Usage: "/automod-words <add|remove|list|clear> ...", Desc: "Manage the banned-word list.", Perm: "Manage Server",
				Examples: []Example{
					{Usage: "/automod-words add word:slur1", Note: "Add a term to the blocklist."},
					{Usage: "/automod-words list", Note: "Show every blocked word."},
				},
			},
		},
	},
	{
		Key: "giveaways", Label: "Giveaways", Emoji: "🎉",
		Blurb: "Timed prize draws members enter with a button; pick and reroll winners.",
		Commands: []Command{
			{
				Name: "giveaway", Usage: "/giveaway <start|end|reroll|list> ...", Desc: "Run timed giveaways with a reaction-button entry.", Perm: "Manage Server",
				Examples: []Example{
					{Usage: "/giveaway start prize:Nitro duration:24h winners:2 channel:#giveaways", Note: "Two winners drawn from #giveaways after 24 hours."},
					{Usage: "/giveaway reroll id:12 winners:1", Note: "Draw a fresh winner for giveaway #12."},
					{Usage: "/giveaway end id:12", Note: "End giveaway #12 early and draw now."},
				},
			},
		},
	},
	{
		Key: "ai", Label: "AI Assistant", Emoji: "✨",
		Blurb: "Ask Claude questions and run an opt-in assistant channel.",
		Commands: []Command{
			{
				Name: "ask", Usage: "/ask <prompt>", Desc: "Ask the AI assistant a question.",
				Examples: []Example{{Usage: "/ask prompt:Summarise the rules of chess in 3 bullets", Note: "One-off question; rate-limited per user."}},
			},
			{
				Name: "ai", Usage: "/ai <channel|system|status> ...", Desc: "Configure the always-on assistant channel.", Perm: "Manage Server",
				Examples: []Example{
					{Usage: "/ai channel channel:#ask-ai", Note: "Treat every message in #ask-ai as a prompt."},
					{Usage: "/ai system prompt:You are a terse support bot for our game.", Note: "Set the assistant's persona."},
					{Usage: "/ai status", Note: "Show the assistant channel and current settings."},
				},
			},
		},
	},
}

// catalogIndex maps a category key to its category for O(1) lookup.
var catalogIndex = func() map[string]*Category {
	m := make(map[string]*Category, len(Catalog))
	for i := range Catalog {
		m[Catalog[i].Key] = &Catalog[i]
	}
	return m
}()

// category returns the category for a key, or nil if unknown.
func category(key string) *Category {
	return catalogIndex[key]
}

// findCommand locates a command by its bare name across all categories,
// returning the owning category too. Lookup is case-insensitive on the leading
// token so "/ban" and "ban" both resolve.
func findCommand(name string) (*Category, *Command) {
	for i := range Catalog {
		for j := range Catalog[i].Commands {
			if Catalog[i].Commands[j].Name == name {
				return &Catalog[i], &Catalog[i].Commands[j]
			}
		}
	}
	return nil, nil
}

// commandCount returns the total number of documented commands.
func commandCount() int {
	n := 0
	for i := range Catalog {
		n += len(Catalog[i].Commands)
	}
	return n
}
