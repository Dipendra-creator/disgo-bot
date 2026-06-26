# disgo-bot — Progress

_Last updated: 2026-06-26_

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

- leveling, economy (non-gambling), verification, logging, automod, giveaways, AI assistant.
- Cross-cutting: Redis Streams workers, full RBAC engine, gateway sharding,
  REST/web dashboard + OAuth2.

The **next module** has not been chosen — confirm with the maintainer before
starting one (the spec mandates incremental, one-module-at-a-time delivery).

## How to verify locally

```bash
make check          # gofmt + go vet + go test -race
make build          # static binary -> bin/disgo
go test ./...       # unit tests
```

Full e2e needs a Discord token + Postgres (see `README.md` → Run / Docker).
