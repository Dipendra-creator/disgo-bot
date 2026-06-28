# disgo-bot — prebuilt deployment branch

This branch carries **only the compiled bot and an installer** — no source. It is
meant for deploying disgo-bot on an Amazon EC2 instance (or any Ubuntu VM) without
a Go toolchain or a source checkout.

The binary is fully self-contained: the dashboard's HTML/CSS/JS assets and all SQL
migrations are embedded. Migrations run automatically on first start.

## Contents

| File | Purpose |
|------|---------|
| `.env.example` | Template — copy to `.env`, fill in, the installer reads it |
| `setup.sh` | Installer — reads `.env` (no prompts), else an interactive wizard |
| `disgo-bot-linux-amd64` | Static binary for x86_64 (Intel/AMD EC2) |
| `disgo-bot-linux-arm64` | Static binary for aarch64 (Graviton EC2) |

`setup.sh` auto-detects the CPU architecture and picks the right binary.

## Quick start (fresh Ubuntu 22.04/24.04 VM)

```bash
git clone -b build https://github.com/Dipendra-creator/disgo-bot.git
cd disgo-bot
cp .env.example .env
nano .env            # fill in your values
sudo ./setup.sh      # reads ./.env — no prompts
```

The installer reads `./.env` (or `--env PATH`, or an existing
`/etc/disgo-bot/disgo-bot.env`) and runs **non-interactively**. With no env file
present it falls back to an interactive wizard. Either way it then:

1. installs `ca-certificates`, `curl`, `openssl`, and — unless you point it at an
   existing database — `postgresql`;
2. provisions a PostgreSQL role + database (or uses the `DATABASE_URL` you set, e.g.
   an RDS endpoint), waiting for the cluster to come up on a fresh VM;
3. installs the arch-matched binary to `/usr/local/bin/disgo-bot`;
4. writes secrets to `/etc/disgo-bot/disgo-bot.env` (mode `0600`, root-owned);
5. creates a hardened `systemd` unit running as an unprivileged `disgo` user, then
   enables and starts it.

The `.env` parser strips a Windows CR, surrounding quotes and stray whitespace —
so a file edited on Windows won't smuggle a trailing `\r` into your bot token
(that is exactly what Discord rejects as `close 4004: Authentication failed`).

## Configuration (`.env`)

| Key | Notes |
|-----|-------|
| `DISCORD_APP_ID` | required |
| `DISCORD_TOKEN` | required — must be current and belong to the same app as the ID |
| `DISCORD_DEV_GUILD_ID` | blank = register commands globally |
| `DISCORD_PRIVILEGED_INTENTS` | `true`/`false`; also toggle in the Discord portal |
| `DATABASE_URL` | set for a remote/managed DB; leave blank to use a local PG |
| `PG_INSTALL` / `PG_DB` / `PG_USER` / `PG_PASSWORD` | local-PG provisioning (blank password = auto-generate) |
| `WEB_ENABLED` / `WEB_ADDR` / `WEB_PUBLIC_URL` / `WEB_COOKIE_SECURE` | dashboard; URL blank = auto-detect public IP |
| `DISCORD_CLIENT_SECRET` | required when `WEB_ENABLED=true` |
| `AI_ENABLED` / `ANTHROPIC_API_KEY` / `AI_MODEL` | optional AI module |
| `DISGO_ENV` / `LOG_LEVEL` / `LOG_FORMAT` | default production / info / json |

## After install

```bash
systemctl status disgo-bot      # state
journalctl -u disgo-bot -f      # live logs
systemctl restart disgo-bot     # apply config edits
sudoedit /etc/disgo-bot/disgo-bot.env   # change config
```

### Ports

| Service | Default | Note |
|---------|---------|------|
| Dashboard | `:8081` | open in the EC2 security group; set `WEB_PUBLIC_URL` to match |
| Health | `:8080/healthz` | liveness probe |
| Metrics | `:9090/metrics` | Prometheus |

> **Dashboard login:** register `${WEB_PUBLIC_URL}/auth/callback` as an OAuth2
> redirect URI in the Discord developer portal, or login will fail. For production
> put the dashboard behind HTTPS (reverse proxy or ALB) and set
> `WEB_PUBLIC_URL=https://…` with `WEB_COOKIE_SECURE=true`.

## Update

Pull the new build and re-run — existing config is reused as defaults, the binary
is swapped, and the service restarts:

```bash
git pull
sudo ./setup.sh
```

## Uninstall

```bash
sudo ./setup.sh uninstall   # stops + removes the service and binary
```

Config (`/etc/disgo-bot`), data (`/var/lib/disgo-bot`) and any local PostgreSQL are
left in place; remove them manually if wanted.

## How this branch is produced

Built from `master` with a static, stripped cross-compile:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o disgo-bot-linux-amd64 ./cmd/bot
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o disgo-bot-linux-arm64 ./cmd/bot
```
