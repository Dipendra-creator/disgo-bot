# disgo-bot — Progress

_Last updated: 2026-06-27_

A production-grade, modular, multipurpose Discord bot in Go, built foundation-first
on Clean Architecture with a plugin-style module system and modern Discord-native
UI (**Components v2**). Explicit non-goals: **not a music bot**, **no gambling /
casino features**.

- **Module path:** `github.com/dipu-sharma/disgo-bot`
- **Go:** 1.26
- **discordgo:** `@master` (`v0.29.1-0.20260214123928-f43dd94faaac`) — pinned for Components v2

## Status at a glance

| Area | State |
| --- | --- |
| Core framework / DI / router | ✅ Shipped |
| Database (Bun/Postgres) + migrations | ✅ Shipped |
| Cache (Redis + in-memory fallback) | ✅ Shipped |
| Config (YAML + env + validation) | ✅ Shipped |
| Observability (zap, Prometheus, Sentry, health) | ✅ Shipped |
| UI library (embeds, buttons, Components v2, paginator) | ✅ Shipped |
| **utility** module | ✅ Shipped |
| **moderation** module | ✅ Shipped |
| **tickets** module | ✅ Shipped |
| **logging** module | ✅ Shipped |
| **leveling** module | ✅ Shipped |
| **economy** module (non-gambling) | ✅ Shipped |
| **verification** module | ✅ Shipped |
| **automod** module | ✅ Shipped |
| **giveaways** module | ✅ Shipped |
| **ai** assistant module | ✅ Shipped |
| Deployment (Docker, compose, k8s, CI) | ✅ Authored |

**Verification (this environment — no Docker / no live token):**
`gofmt -l .` clean · `go build ./...` ✅ · `go vet ./...` ✅ ·
`go test ./...` ✅ · `go test -race -count=1 ./...` ✅ · `go mod tidy` stable.

Live Discord end-to-end (clicking buttons, real guild) is **not** exercised here —
that requires a bot token + Postgres on the user's machine.

## Shipped modules

### Foundation (Phase 1)

- `cmd/bot` — entrypoint: config → deps → modules → run → graceful shutdown.
- `internal/config` — typed config, three-layer merge (defaults → YAML → env), validator.
- `internal/logger` — zap structured logger.
- `internal/observability` — Prometheus metrics, `/healthz` + `/readyz`, Sentry init.
- `internal/database` — `bun.DB` over `pgdriver`, pool, embedded `//go:embed` migration runner.
- `internal/cache` — `Cache` interface, Redis impl, in-memory fallback.
- `internal/bot` — session lifecycle, intents, dispatch wiring.
- `internal/router` — interaction registry + dispatcher + command sync (bulk overwrite).
- `internal/ui` — embeds, buttons, Components v2 builders, paginator, state helpers, theme.
- `shared` — `Module` interface, `Deps` DI container, `Command`, `Context`, permissions, custom-ID codec, errors.
- `pkg` — `snowflake`, `humanize`, `duration` helpers.
- `modules/utility` — `/ping`, `/serverinfo` (+ refresh button), `/userinfo` (+ context menu), `/avatar` (size buttons).

### moderation (`modules/moderation`, migration `0002_moderation.sql`)

Numbered **cases**, optional DM-on-action, configurable **mod-log** channel.
Role-hierarchy + identity (self/owner/bot) guards on every action; commands gated
by `DefaultMemberPermissions` **and** a runtime permission re-check.

Commands: `/ban` (perm + temp, auto-unban sweeper), `/unban`, `/kick`,
`/timeout` (≤28d), `/untimeout`, `/warn`, `/warnings` (paginated), `/case`,
`/reason`, `/delwarn`, `/purge` (≤14-day bulk delete), `/modlog`.

Durations: compact `10m` / `2h30m` / `7d` / `1w` (`pkg/duration`). Temp-bans
reconciled by an in-process sweeper (60s tick) that lifts expired bans.

### tickets (`modules/tickets`, migration `0003_tickets.sql`)

Admin posts a **panel** (Components v2); users click **Open Ticket** → private
channel (hidden from `@everyone`, visible to opener + staff role). Staff **claim**
/ **close** via buttons; close posts a `.txt` **transcript** to the log channel and
deletes the channel. One open ticket per user, enforced by a partial unique index.

Commands: `/ticket-setup` (category / staff role / log channel), `/ticket-panel`,
`/close`, `/ticket-add`, `/ticket-remove`.

Channel creation is rollback-safe (DB insert failure → `ChannelDelete`).
Interaction responses are always sent **before** the channel is deleted.

### logging (`modules/logging`, migration `0004_logging.sql`)

Mirrors gateway events into per-category log channels. Categories: `message`
(edits/deletions with before/after content), `member` (joins/leaves), `server`
(bans/unbans, channel + role create/delete). Configured via `/logging
set|disable|status` (Manage Server).

Unlike other modules this consumes **gateway events** (`Module.Events()`), not
interactions — handlers run in discordgo's goroutines outside the router, so each
is wrapped in a panic-recover guard. Per-guild settings are cached in-process
(events fire hot) and invalidated on change. `member` events and message
**content** need the privileged intents (`discord.privileged_intents`), which
also enable a 200-message-per-channel state cache so edit/delete logs carry prior
content.

### leveling (`modules/leveling`, migration `0005_leveling.sql`)

XP-and-ranks system. Members earn `xp_min`–`xp_max` XP per message, gated by a
per-user cooldown enforced in the **cache** (`lvl:cd:<guild>:<user>` with TTL) so
the hot path avoids a DB write on every message. Cumulative XP maps to a level
via the fixed `5·L²+50·L+100` curve (`levels.go`). On level-up the module grants
reward roles (stacked or highest-only) and announces (configurable channel).
Commands: `/rank`, `/leaderboard` (paginated), `/level-config`, `/level-role`,
`/xp` (admin). Atomic XP increment via `INSERT … ON CONFLICT … RETURNING`;
settings cached in-process and invalidated on change.

### economy (`modules/economy`, migration `0006_economy.sql`)

Per-guild virtual currency — **non-gambling by design** (no betting, slots or
chance mechanics). Members earn from `/daily` (fixed 24h cooldown) and `/work`
(randomised reward, configurable cooldown), hold funds in a **wallet** and a
**bank**, transfer to each other, and spend in a per-guild **shop** that can grant
a role on purchase. Items support limited or unlimited stock and an inventory.

Commands: `/balance`, `/daily`, `/work`, `/pay`, `/deposit`, `/withdraw`,
`/shop` (paginated), `/buy`, `/inventory`, `/rich` (paginated leaderboard),
`/eco-config` (currency/daily/work/starting — Manage Server), `/eco-admin`
(give/set/reset balances, shop-add/shop-remove — Manage Server).

Every money mutation is **atomic and race-safe**: transfers, deposits, withdrawals
and purchases run `UPDATE … WHERE balance >= amount RETURNING` (→ `ErrInsufficient`
on a no-row), limited stock decrements via a guarded `RETURNING` (→ `ErrOutOfStock`),
and buys run inside `RunInTx`. Periodic earnings stamp `last_daily`/`last_work` in
the same guarded update (cutoff check), so a member on cooldown causes no write.
Settings cached in-process and invalidated on change.

### verification (`modules/verification`, migration `0007_verification.sql`)

A member gate. An admin configures a **verified role** (plus optional log channel,
panel message and button label) with `/verify-setup` and posts a Components-v2
panel with `/verify-panel`; members click the button to receive the role. The
grant is **idempotent** — members who already hold the role get an "already
verified" reply, and `GuildMemberRoleAdd` is a no-op on Discord's side regardless.
First-time verifications are written to `verification_records`
(`INSERT … ON CONFLICT DO NOTHING`, `RowsAffected` ⇒ first-time) and, when only
new, mirrored to the log channel. A failed grant (role hierarchy / missing Manage
Roles) surfaces a friendly message and is logged. `/verify-disable` flips the
enabled flag without discarding config; `/verify-status` shows the configuration
and verified count. Settings cached in-process and invalidated on change.

### automod (`modules/automod`, migration `0008_automod.sql`)

Automatic content moderation driven by gateway **message events** (like logging,
each handler is panic-guarded). Four independent filters — **banned words**,
**invite links**, **mass mentions**, **spam** — each with its own enable flag and
**action** (`delete`, or `delete` + **timeout**). On a match the message is
deleted, the author is optionally timed out (`GuildMemberTimeout`, duration
configurable up to Discord's 28-day ceiling) and the action is mirrored to a log
channel. A configurable role and anyone with **Manage Messages** are exempt.

Filters evaluate in priority order (words → invites → mentions → spam); the first
hit wins. Word matching tokenises content so single terms match whole words
(no "class" → "ass" false positives) while multi-word phrases match as a
substring. Spam uses an **in-process sliding window** of recent message times per
member (its own mutex), avoiding a DB/cache round-trip on the hot path. Per-guild
settings and the banned-word set are cached in-process and invalidated on change.

Configured via `/automod` (status, log, exempt, timeout, and a per-filter toggle
subcommand each) and `/automod-words` (add/remove/list/clear) — both Manage
Server. Inspecting message content needs the privileged `MessageContent` intent.

### giveaways (`modules/giveaways`, migration `0009_giveaways.sql`)

Timed prize draws. `/giveaway start` posts a Components-v2 panel with an **Enter**
button whose label shows the live entry count; clicking toggles entry (a second
click leaves). An **in-process sweeper** (20s tick, mirroring moderation's
temp-ban sweeper) ends giveaways whose `ends_at` has passed: it draws winners
uniformly at random (`math/rand` shuffle), records them, edits the panel to the
ended state and pings the winners in the channel (allowed-mentions scoped to
users). `/giveaway end` ends one early; `/giveaway reroll` draws fresh winners
for an ended giveaway; `/giveaway list` shows active ones — all Manage Server.

The panel post happens after the row insert (the button needs the giveaway ID);
a failed post rolls back the row. Entries cascade-delete with the giveaway via a
foreign key. Because end time lives in the database, giveaways still resolve
after a restart — no external scheduler.

### ai (`modules/ai`, migration `0010_ai.sql`)

An opt-in Claude-backed assistant. `/ask` answers a one-off question for anyone
(rate-limited per user via the cache, `ai:cd:<guild>:<user>` with a 12s TTL);
admins can set an opt-in **assistant channel** where the bot replies to every
message (driven by a panic-guarded `MessageCreate` handler, like logging/automod).

Completions go through a **`Provider` interface** so the backend stays swappable
and mockable; the default `anthropicProvider` is a dependency-free HTTP client for
the Anthropic **Messages API** (`POST /v1/messages`, `x-api-key` +
`anthropic-version: 2023-06-01`, request `{model, max_tokens, system, messages}`,
response `content[].text`). The module is **inert until an API key is supplied** —
`config.AI.Ready()` (`AI_ENABLED` + `ANTHROPIC_API_KEY`) gates provider wiring, and
`/ask` / the assistant channel report unavailability rather than failing hard.

Per-guild **system prompt** override and assistant channel are stored in
`ai_settings` and cached in-process (invalidated on change). Replies are
rune-safe-truncated under Discord's 2000-char limit. Default model
`claude-opus-4-8` (override `AI_MODEL`); the assistant channel needs the
privileged `MessageContent` intent.

Configured via `/ask` (public) and `/ai channel|system|status` (Manage Server).

## Conventions worth knowing

- **Custom-ID routing:** `module:action:arg1:arg2`, encoded/decoded by
  `shared.BuildID` / `shared.ParseID`; the dispatcher routes by `module` prefix
  and hands `action` + args to the handler via `Context.Args`.
- **Discord IDs** are stored as `BIGINT` (`int64`); each module has `pid(string)→int64`
  and `sid(int64)→string` helpers.
- **Per-guild counters** (case numbers, ticket numbers) are atomic via
  `INSERT … ON CONFLICT … DO UPDATE … RETURNING` inside a transaction.
- **Permissions:** defense in depth — Discord hides commands via
  `DefaultMemberPermissions`, and handlers re-check at runtime
  (`shared.RequirePermission`) plus role-hierarchy checks.
- Every dispatch is wrapped with timeout, panic recovery, Prometheus metrics and
  structured logging.

## Roadmap (not yet started)

Built incrementally on the same foundation, module by module:

- Cross-cutting: Redis Streams workers, full RBAC engine, gateway sharding,
  REST/web dashboard + OAuth2.

All originally-scoped feature modules are shipped. Modules were built in sequence
(logging → leveling → economy → verification → automod → giveaways → AI), each
verified and committed independently.

## How to verify locally

```bash
make check          # gofmt + go vet + go test -race
make build          # static binary -> bin/disgo
go test ./...       # unit tests
```

Full e2e needs a Discord token + Postgres (see `README.md` → Run / Docker).
