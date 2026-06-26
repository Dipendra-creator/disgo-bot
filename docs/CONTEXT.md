# disgo-bot — Build Context & Decisions

This document captures the *why* behind the project: the original intent, the
decisions made, the conventions to keep, and how to pick the work back up. It is a
handoff/continuity note, not API docs (see `README.md`) or status (see `PROGRESS.md`).

## Original intent

Build a **production-grade, modular, multipurpose Discord bot in Go** to compete
with Dyno / Carl-bot / Sapphire / MEE6 / Ticket Tool, using modern Discord-native
UI (**Components v2**).

Hard constraints from the spec:

- **Do NOT build a music bot.**
- **Do NOT create casino / gambling games.**
- **Clean Architecture**; each feature is an independent, plugin-style module.
- **Incremental delivery** — _"Do NOT generate everything at once. Implement the
  project incrementally, module by module, beginning with the core framework,
  shared libraries, interaction framework, database layer, and configuration
  system before building feature modules."_
- Docker-first infrastructure.

## Key decisions

| Decision | Choice | Rationale |
| --- | --- | --- |
| Language / toolchain | Go 1.26 | Spec; static binary, distroless image |
| Discord library | `discordgo@master` | Only master has Components v2 (Container/Section/TextDisplay/Separator/Thumbnail) |
| ORM | Bun over Postgres (`pgdriver`/`pgdialect`/`bundebug`) | Lightweight, SQL-first, good for explicit migrations |
| Migrations | Embedded `//go:embed` SQL, run on boot | Single binary ships its own schema |
| Cache | `Cache` interface; Redis impl + in-memory fallback | Bot runs in dev without Redis |
| Logging / metrics / errors | zap / Prometheus / Sentry | Production observability baseline |
| Config | YAML + env overrides + `validator` | 3-layer merge; secrets via env |
| DI | `Deps` container injected into `Module.Init` | No globals, interface-driven |
| Discord IDs in DB | `BIGINT` (`int64`) | Compact, indexable; `pid`/`sid` convert at the edges |

## Architecture seams

- **Module contract** (`shared/module.go`): `Name() / Init(*Deps) / Commands() /
  Components() / Modals() / Events()`. Every feature implements this; register it
  in `cmd/bot/main.go`'s module slice.
- **Router** (`internal/router`): one `InteractionCreate` handler dispatches by type —
  commands by name, components/modals by the `module:action:args` custom-ID convention,
  autocomplete by the command's fn. Middleware: panic recovery, metrics, logging, timeout.
- **UI** (`internal/ui`): theme + reusable builders; Components v2 builders gated behind
  `MessageFlagsIsComponentsV2`; `Paginator` encodes its state in the custom ID.

## Conventions to keep (so new modules match the existing ones)

1. **Custom IDs:** `shared.BuildID(module, action, args...)` /
   `shared.ParseID`; never hand-format custom IDs.
2. **ID conversion:** per-module `pid(string) int64` / `sid(int64) string`; store
   `int64`, convert only at Discord/DB boundaries.
3. **Atomic per-guild counters:** `INSERT … ON CONFLICT … DO UPDATE … RETURNING`
   inside `RunInTx` (see `moderation`/`tickets` repos).
4. **Defense-in-depth permissions:** set `DefaultMemberPermissions` on the command
   def **and** re-check at runtime with `shared.RequirePermission`; add
   role-hierarchy / identity checks where actions target members.
5. **Respond before destroying:** when a handler deletes the channel it's running in
   (ticket close), send the interaction response **first** — you can't respond into a
   deleted channel.
6. **Rollback side effects:** if a Discord resource is created then a DB write fails,
   delete the resource (ticket channel) so state stays consistent.
7. **Errors:** return `shared.UserErr(...)` for user-facing messages; the dispatcher
   renders them. Repos return typed sentinels (`ErrCaseNotFound`, `ErrNoTicket`).
8. **Verify discordgo symbols against the vendored master source** before using them —
   master is bleeding-edge and field names can differ from any docs. The module cache
   is at `/root/go/pkg/mod/github.com/bwmarrin/discordgo@v0.29.1-0.20260214123928-f43dd94faaac`.
9. **Library docs:** fetch current docs via Context7 MCP (`/bwmarrin/discordgo`) when
   working with discordgo APIs, per the maintainer's global rule.

## Verification gate (run before considering a module done)

```bash
gofmt -l .                       # must be empty
go build ./...                   # exit 0
go vet ./...                     # exit 0
go test ./...                    # all pass
go test -race -count=1 ./...     # race-clean
go mod tidy                      # go.mod / go.sum unchanged
```

## Things that bit us (and the fix)

- `IntentsGuildModeration` does not exist in this discordgo master → use
  `IntentsGuildBans`.
- `pkg/duration` `Human()` normalizes `7d` → `1w`; tests must expect the normalized form.
- The `unused` linter is strict: remove dead structs/funcs/imports
  (`caseCounter`, `withinCloseGrace` + its `time` import were dropped).

## How to continue

1. Pick the next module **with maintainer confirmation** (incremental delivery is a
   hard constraint). Candidates: leveling, economy (non-gambling), verification,
   logging, automod, giveaways, AI assistant.
2. Scaffold under `modules/<name>/` mirroring `tickets`/`moderation`
   (`module.go`, `models.go`, `repository.go`, `service.go`, command/component files,
   `ids.go`, `options.go`, `<name>_test.go`).
3. Add a migration `database/migrations/000N_<name>.sql`.
4. Register the module in `cmd/bot/main.go`.
5. Run the verification gate above; update `README.md` + `docs/PROGRESS.md`.

## Communication style

The maintainer works in **caveman mode** (terse: drop articles/filler/hedging,
keep all technical substance and exact identifiers). Code, commits, PRs and security
warnings are written in normal prose. Toggle off with "stop caveman" / "normal mode".
