# disgo-bot

A production-grade, modular, multipurpose Discord bot in Go — built on a clean,
plugin-style architecture and modern Discord-native UI (**Components v2**).

This repository is **Phase 1**: the foundation (core framework, interaction
router, database, cache, config, observability, UI library) plus one working
vertical-slice feature module (`utility`) that proves the framework boots and
serves real commands end-to-end. Feature modules (moderation, tickets, leveling,
economy, verification, logging, automod, giveaways, AI, …) are added
incrementally on top of this foundation.

> Not a music bot. No gambling/casino features.

> **Docs:** [`docs/PROGRESS.md`](docs/PROGRESS.md) (what's shipped + verification
> state) · [`docs/CONTEXT.md`](docs/CONTEXT.md) (decisions, conventions, how to
> continue).

## Features

### Help

| Command | What it shows |
| --- | --- |
| `/help` | Browsable, **Components v2** guide to every command — overview with a category select menu, per-category command lists, and per-command examples |
| `/help command:<name>` | Jumps straight to one command's full examples; the `command` option **autocompletes** across the whole catalog |

The help module is self-contained (it imports no other module) and ships a
hand-authored catalog of every command with real-world examples, so the modules
stay independent while the guide stays accurate.

### Utility

| Command | What it shows |
| --- | --- |
| `/ping` | Gateway heartbeat + response round-trip latency (deferred reply) |
| `/serverinfo` | Guild stats + a **Components v2** container and a refresh button (component routing) |
| `/userinfo [user]` | Member card; also a **User context-menu** command |
| `/avatar [user]` | Avatar embed with size buttons (256/1024/4096) + open link |

### Moderation

Every action is recorded as a numbered **case**, optionally DMed to the user and
mirrored to a configurable **mod-log** channel. Role-hierarchy and identity
(self/owner/bot) checks guard every action, and commands are gated by Discord
permissions (`DefaultMemberPermissions` + a runtime re-check).

| Command | Permission | Purpose |
| --- | --- | --- |
| `/ban <user> [reason] [delete_days] [duration]` | Ban Members | Permanent or temporary ban (auto-unban on expiry) |
| `/unban <user_id> [reason]` | Ban Members | Lift a ban by ID |
| `/kick <user> [reason]` | Kick Members | Remove a member |
| `/timeout <user> <duration> [reason]` | Moderate Members | Discord timeout (max 28 days) |
| `/untimeout <user> [reason]` | Moderate Members | Clear a timeout |
| `/warn <user> <reason>` | Moderate Members | Log a warning |
| `/warnings <user>` | Moderate Members | Paginated active warnings |
| `/case <number>` | Moderate Members | View a case |
| `/reason <number> <reason>` | Moderate Members | Edit a case's reason |
| `/delwarn <number>` | Moderate Members | Deactivate a warning |
| `/purge <count> [user]` | Manage Messages | Bulk-delete recent messages (≤14 days) |
| `/modlog <channel>` | Manage Server | Set the mod-log channel |

Durations use a compact format: `10m`, `2h30m`, `7d`, `1w`. Temp-bans are
reconciled by an in-process sweeper that lifts expired bans every minute.

### Tickets

Admins post a **panel**; users click **Open Ticket** to get a private channel
(hidden from `@everyone`, visible to the opener and the staff role). Staff
**claim** and **close** via buttons; on close a `.txt` **transcript** is posted
to the log channel and the ticket channel is deleted. One open ticket per user.

| Command | Permission | Purpose |
| --- | --- | --- |
| `/ticket-setup <category> [staff_role] [log_channel]` | Manage Server | Configure where tickets/transcripts go |
| `/ticket-panel [channel] [title] [description]` | Manage Server | Post a Components-v2 panel with an Open button |
| `/close [reason]` | Staff / opener | Close the current ticket |
| `/ticket-add <user>` | Staff / opener | Grant a user access to the ticket |
| `/ticket-remove <user>` | Staff | Revoke a user's access |

Buttons: **Open Ticket** (panel) → **Claim** / **Close** (in-channel) → close
shows an ephemeral **Confirm/Cancel** prompt before deleting.

### Logging

Mirrors gateway events into per-category log channels. Configure with one
command; each category routes to its own channel (or is disabled).

| Category | Events |
| --- | --- |
| `message` | message edits and deletions (with before/after content) |
| `member` | member joins and leaves |
| `server` | bans/unbans, channel create/delete, role create/delete |

| Command | Permission | Purpose |
| --- | --- | --- |
| `/logging set <category> <channel>` | Manage Server | Route a category to a channel |
| `/logging disable <category>` | Manage Server | Stop logging a category |
| `/logging status` | Manage Server | Show the current configuration |

> **Privileged intents:** `member` events and message **content** require the
> `GuildMembers` and `MessageContent` intents. Set `discord.privileged_intents:
> true` (or `DISCORD_PRIVILEGED_INTENTS=true`) **and** enable them in the Discord
> developer portal. Other events (bans, channels, roles) work without them.

### Leveling

Members earn XP for chatting (rate-limited per user via a cache-backed
cooldown), level up along the classic `5·L² + 50·L + 100` curve, and can earn
reward roles. Level-ups are announced; rank cards and a paginated leaderboard
render with a progress bar.

| Command | Permission | Purpose |
| --- | --- | --- |
| `/rank [user]` | — | Show level, rank and progress to the next level |
| `/leaderboard` | — | Paginated server XP leaderboard |
| `/level-config <sub>` | Manage Server | enable/disable, cooldown, xp-range, announce, stack |
| `/level-role add\|remove\|list` | Manage Roles | Map levels to reward roles |
| `/xp give\|set\|reset` | Manage Server | Adjust a member's XP |

Reward roles can **stack** (keep every earned role) or keep only the highest.
XP gain needs only `GuildMessages` — no privileged intents.

### Economy

A per-guild virtual currency — **non-gambling by design** (no betting, slots or
chance mechanics). Members earn from `/daily` and `/work`, hold funds in a wallet
and a bank, transfer to each other, and spend in a configurable shop that can
grant roles on purchase.

| Command | Permission | Purpose |
| --- | --- | --- |
| `/balance [user]` | — | Wallet, bank and net worth |
| `/daily` | — | Claim the once-per-day reward |
| `/work` | — | Earn a randomised reward on a cooldown |
| `/pay <user> <amount>` | — | Send currency from your wallet |
| `/deposit <amount>` | — | Move wallet → bank |
| `/withdraw <amount>` | — | Move bank → wallet |
| `/shop` | — | Browse the paginated shop |
| `/buy <item>` | — | Purchase an item (grants its role, if any) |
| `/inventory [user]` | — | List owned items |
| `/rich` | — | Paginated net-worth leaderboard |
| `/eco-config <sub>` | Manage Server | currency, daily, work, starting balance |
| `/eco-admin <sub>` | Manage Server | give/set/reset balances, shop-add/shop-remove |

All money operations are **atomic and guarded** — transfers, deposits, withdrawals
and purchases use `UPDATE … WHERE balance >= amount RETURNING` (and stock-guarded
`RETURNING` for limited items) so concurrent commands can't overspend or oversell.
Periodic earnings are gated by stamped cooldown columns checked in the same
atomic update.

### Verification

A member gate. An admin picks a **verified role** and posts a **panel**; members
click one button to receive the role. Grants are **idempotent** (clicking again
is a no-op) and each member's first verification is audited and optionally
mirrored to a log channel.

| Command | Permission | Purpose |
| --- | --- | --- |
| `/verify-setup <role> [log_channel] [message] [button_label]` | Manage Server | Configure and enable the gate |
| `/verify-panel [channel] [title] [description]` | Manage Server | Post a Components-v2 panel with the verify button |
| `/verify-disable` | Manage Server | Stop the button granting the role (config kept) |
| `/verify-status` | Manage Server | Show the current configuration and verified count |

Needs the bot's role to sit **above** the verified role and to hold **Manage
Roles**; a friendly error is shown if the grant is refused.

### AutoMod

Automatic content moderation. Four independent filters, each with its own
on/off toggle and **action** — *delete the message*, or *delete and timeout the
author*. Offending messages are always removed and optionally mirrored to a log
channel. A configurable role and anyone with **Manage Messages** are exempt.

| Filter | Trips on |
| --- | --- |
| Banned words | A configurable word/phrase list (whole-word match for single terms) |
| Invite links | `discord.gg` / `discord.com/invite` links |
| Mass mentions | A message with ≥ N user/role mentions (incl. `@everyone`) |
| Spam | More than N messages from one member within a rolling window |

| Command | Permission | Purpose |
| --- | --- | --- |
| `/automod status` | Manage Server | Show the full configuration |
| `/automod log [channel]` | Manage Server | Set/clear the action-log channel |
| `/automod exempt [role]` | Manage Server | Set/clear a bypass role |
| `/automod timeout <seconds>` | Manage Server | Duration used by the timeout action |
| `/automod words\|invites\|mentions\|spam <enabled> [action] [thresholds]` | Manage Server | Toggle and tune a filter |
| `/automod-words add\|remove\|list\|clear` | Manage Server | Manage the banned-word list |

> **Privileged intent:** inspecting message **content** (words/invites/mentions)
> requires the `MessageContent` intent — set `discord.privileged_intents: true`
> and enable it in the Discord developer portal.

### Giveaways

Timed prize draws. A host starts one with a prize, duration and winner count;
members **enter with one button** (click again to leave); an in-process sweeper
**draws winners** when the timer expires, edits the panel to show the result and
pings the winners. Winners can be **rerolled** afterwards.

| Command | Permission | Purpose |
| --- | --- | --- |
| `/giveaway start <prize> <duration> [winners] [channel]` | Manage Server | Post a giveaway (durations like `30m`, `2h`, `7d`) |
| `/giveaway end <id>` | Manage Server | End a running giveaway now and draw winners |
| `/giveaway reroll <id> [winners]` | Manage Server | Draw fresh winners for an ended giveaway |
| `/giveaway list` | Manage Server | List the running giveaways |

The entry button's label shows the live entry count. The end-time sweeper runs
every 20s, so giveaways still resolve after a restart — no external scheduler.

### AI Assistant

An opt-in Claude-backed assistant. Anyone can `/ask` a one-off question
(rate-limited per user); admins can dedicate a **channel** where the bot replies
to every message. Completions go through a `Provider` interface — an Anthropic
**Messages API** implementation by default — so the backend is swappable and
mockable. The module is **inert until the bot owner supplies an API key**;
without one the commands report that the assistant is unavailable rather than
failing.

| Command | Permission | Purpose |
| --- | --- | --- |
| `/ask <prompt>` | — | Ask a one-off question (public, rate-limited) |
| `/ai channel [channel]` | Manage Server | Set/clear an opt-in channel the bot replies in |
| `/ai system [prompt]` | Manage Server | Set/clear a custom system prompt (omit to reset) |
| `/ai status` | Manage Server | Show configuration and availability |

Configure with `AI_ENABLED=true` + `ANTHROPIC_API_KEY` (model defaults to
`claude-opus-4-8`; override with `AI_MODEL`). The assistant **channel** reads
message content, so it needs the `MessageContent` privileged intent.

## Architecture

Clean Architecture with an interface-driven module plugin system. Each feature
is an independent module implementing one contract:

```go
type Module interface {
    Name() string
    Init(*Deps) error
    Commands() []*Command              // slash + context-menu defs & handlers
    Components() map[string]HandlerFunc // customID action -> handler
    Modals() map[string]HandlerFunc     // customID action -> handler
    Events() []interface{}              // raw gateway handlers
}
```

Dependencies are constructed once in `main` and injected via a `Deps` container
(no globals): `Config`, `Log` (zap), `DB` (Bun/Postgres), `Cache`,
`Session` (discordgo), `Metrics`.

Interactions route through a single dispatcher. Slash/context commands route by
name; components and modals route by a custom-ID convention:

```
module:action:arg1:arg2
```

`shared.BuildID` / `shared.ParseID` encode and decode these; the dispatcher
hands `action` + args to the handler via `Context.Args`. Every dispatch is
wrapped with timeout, panic recovery, Prometheus metrics and structured logging.

### Layout

```
cmd/bot/         entrypoint: config -> deps -> modules -> run -> graceful shutdown
internal/
  config/        typed config, YAML + env load, validation
  logger/        zap structured logger
  observability/ prometheus metrics, /healthz + /readyz, sentry init
  database/      bun.DB over pgdriver, pool, embedded migration runner
  cache/         Cache interface, redis impl, in-memory fallback
  bot/           session lifecycle, intents, dispatch wiring
  router/        interaction registry + dispatcher + command sync
  web/           optional dashboard: Discord OAuth2, sessions, per-guild config API
  ui/            embeds, buttons, Components v2 builders, paginator, states, theme
modules/
  utility/       ping/serverinfo/userinfo/avatar
  moderation/    cases, ban/kick/timeout/warn/purge, mod-log
  tickets/       panel, private channels, claim/close, transcripts
  logging/       gateway event mirror (message/member/server)
  leveling/      XP, ranks, reward roles, leaderboard
  economy/       non-gambling currency, wallet/bank, shop, inventory
  verification/  member gate: verify button grants a role, audited
  automod/       content filters: words, invites, mentions, spam
  giveaways/     timed prize draws: entry button, sweeper, reroll
  ai/            opt-in Claude assistant: /ask + assistant channel, provider iface
shared/          Module interface, Deps, Command, Context, permissions, errors, customid, Configurable
pkg/             exported helpers (snowflake, humanize)
database/        //go:embed migrations
deployments/     Dockerfile, docker-compose, k8s manifests
configs/         config.example.yaml
```

## Requirements

- Go **1.26+**
- PostgreSQL **18** (or run via docker-compose)
- Redis **7** (optional — falls back to an in-process memory cache when disabled)
- A Discord application + bot token ([Developer Portal](https://discord.com/developers/applications))

> Uses `discordgo@master` for Components v2 support.

## Configuration

Config is merged from three layers, later overriding earlier:

1. Built-in defaults
2. A YAML file (`DISGO_CONFIG` env, else `./config.yaml`)
3. Environment variables (secrets and common overrides)

```bash
cp configs/config.example.yaml config.yaml   # then fill in token + app_id
# or configure entirely via env:
cp .env.example .env
```

Key env vars: `DISCORD_TOKEN`, `DISCORD_APP_ID`, `DISCORD_DEV_GUILD_ID`
(guild-scoped commands register instantly in dev), `DATABASE_URL`,
`REDIS_ENABLED`/`REDIS_ADDR`, `LOG_LEVEL`/`LOG_FORMAT`. Optional web dashboard:
`WEB_ENABLED`, `WEB_PUBLIC_URL`, `DISCORD_CLIENT_SECRET`, `WEB_ADDR`,
`WEB_COOKIE_SECURE`, `WEB_SESSION_HOURS` (see [Web Dashboard](#web-dashboard)).
Full list in `.env.example`.

## Run

### Local

```bash
# Postgres must be reachable (see DATABASE_URL). Redis optional.
make run         # go run ./cmd/bot
```

On startup the bot connects, runs embedded DB migrations, registers commands
(guild-scoped if `DISCORD_DEV_GUILD_ID` is set, else global) and sets presence.

### Docker

```bash
cp .env.example .env   # set DISCORD_TOKEN + DISCORD_APP_ID
make up                # docker compose: Postgres + Redis + bot
```

## Development

```bash
make check   # gofmt + go vet + go test -race
make build   # static binary -> bin/disgo
make lint    # golangci-lint (if installed)
make help    # list all targets
```

## Observability

- Health: `GET :8080/healthz` (liveness), `GET :8080/readyz` (readiness)
- Metrics: `GET :9090/metrics` (Prometheus) — command counts, latency histogram,
  interaction totals
- Errors: optional Sentry (`SENTRY_ENABLED=true` + `SENTRY_DSN`)

## Web Dashboard

An **optional** browser dashboard lets server admins configure modules without
slash commands. It is **disabled by default** and started only when
`WEB_ENABLED=true` and the OAuth credentials are present.

- **Login:** Discord OAuth2 (`identify guilds` scopes) with **state + PKCE
  (S256)**. The OAuth token is used once at the callback to read the user's
  profile and guild list, then **discarded** — never persisted or sent to the
  browser.
- **Sessions:** an opaque 256-bit token in an httpOnly, `SameSite=Lax`,
  `Secure` (configurable) cookie; the server record lives in the shared cache
  with a TTL and is destroyed on logout.
- **Authorization:** every guild route re-checks server-side that the user has
  **Manage Server** on that guild **and** the bot is present — the client's
  guild list is never trusted for writes. Mutations also require an
  origin/referer CSRF check.
- **Config seam:** modules opt in by implementing `shared.Configurable`
  (`ConfigSchema` + `GetConfig`/`SetConfig`). The web layer discovers them by
  type-assertion and renders a form per `ConfigSchema` — it never touches a
  module's tables. Every module with per-guild settings is wired — leveling,
  logging, moderation, tickets, economy, verification, automod, and AI;
  giveaways has no guild-level settings, so it is intentionally not exposed.
- **Audit log:** every accepted config change is recorded (who, which module,
  the submitted fields, when) to `web_audit_log` and surfaced per guild in the
  dashboard. Recording is best-effort — it never fails the config write.

REST API (all under auth; guild routes re-check Manage Server):

| Route | Purpose |
| --- | --- |
| `GET /api/me` | Current user + manageable guilds |
| `GET /api/guilds/{id}/modules` | List configurable modules: schema + values |
| `GET /api/guilds/{id}/modules/{mod}` | One module's schema + values |
| `PATCH /api/guilds/{id}/modules/{mod}` | Validate + apply a partial config patch |
| `GET /api/guilds/{id}/audit` | Recent dashboard config changes (newest first) |

Enable it:

```bash
WEB_ENABLED=true
WEB_PUBLIC_URL=https://dash.example.com   # base for the OAuth redirect
DISCORD_CLIENT_SECRET=...                 # from the Developer Portal
WEB_COOKIE_SECURE=true                    # behind HTTPS in production
```

Add `${WEB_PUBLIC_URL}/auth/callback` as a redirect URI in the Discord
Developer Portal. The dashboard (a minimal embedded static page) is then served
at `WEB_PUBLIC_URL` (listens on `WEB_ADDR`, default `:8081`).

## Adding a module

1. Create `modules/<name>/module.go` with a type embedding `shared.Base` and
   implementing `Name()` + `Init()`.
2. Add commands in `Commands()`; map component/modal handlers in
   `Components()` / `Modals()`; build custom IDs with `shared.BuildID(name, action, args...)`.
3. Register it in `cmd/bot/main.go`'s module slice.

The router, metrics, logging, DB and cache are provided automatically via `Deps`.

## Roadmap

Built incrementally on this foundation. Shipped: utility, **moderation**,
**tickets**, **logging**, **leveling**, **economy**, **verification**,
**automod**, **giveaways**, **AI assistant**, and a **web dashboard**
(Discord OAuth2 + per-guild module config; every settings-bearing module wired).
Next: Redis Streams workers, full RBAC, and gateway sharding.

## License

TBD.
