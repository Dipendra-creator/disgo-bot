# disgo-bot — prebuilt deployment branch

This branch carries **only the compiled bot and an installer** — no source. It is
meant for deploying disgo-bot on an Amazon EC2 instance (or any Ubuntu VM) without
a Go toolchain or a source checkout.

The binary is fully self-contained: the dashboard's HTML/CSS/JS assets and all SQL
migrations are embedded. Migrations run automatically on first start.

## Contents

| File | Purpose |
|------|---------|
| `setup.sh` | Interactive installer — asks for config, then installs a systemd service |
| `disgo-bot-linux-amd64` | Static binary for x86_64 (Intel/AMD EC2) |
| `disgo-bot-linux-arm64` | Static binary for aarch64 (Graviton EC2) |

`setup.sh` auto-detects the CPU architecture and picks the right binary.

## Quick start (fresh Ubuntu 22.04/24.04 VM)

```bash
git clone -b build https://github.com/Dipendra-creator/disgo-bot.git
cd disgo-bot
sudo ./setup.sh
```

The installer **asks for the configuration first** (the same values used locally —
Discord token, app ID, OAuth client secret, database, dashboard URL, etc.), then:

1. installs `ca-certificates`, `curl`, and — if you choose a local database —
   `postgresql`;
2. provisions a PostgreSQL role + database (or uses a `DATABASE_URL` you paste, e.g.
   an RDS endpoint);
3. installs the binary to `/usr/local/bin/disgo-bot`;
4. writes secrets to `/etc/disgo-bot/disgo-bot.env` (mode `0600`, root-owned);
5. creates a hardened `systemd` unit running as an unprivileged `disgo` user, then
   enables and starts it.

## Configuration prompts

| Prompt | Env key | Notes |
|--------|---------|-------|
| Application (client) ID | `DISCORD_APP_ID` | required |
| Bot token | `DISCORD_TOKEN` | required, hidden input |
| Dev guild ID | `DISCORD_DEV_GUILD_ID` | blank = register commands globally |
| Privileged intents | `DISCORD_PRIVILEGED_INTENTS` | also enable in the Discord portal |
| Database | `DATABASE_URL` | install local PG, or paste a remote DSN |
| Dashboard | `WEB_ENABLED` / `WEB_ADDR` / `WEB_PUBLIC_URL` / `WEB_COOKIE_SECURE` | |
| OAuth client secret | `DISCORD_CLIENT_SECRET` | required when the dashboard is on |
| AI module | `AI_ENABLED` / `ANTHROPIC_API_KEY` / `AI_MODEL` | optional |
| Runtime | `DISGO_ENV` / `LOG_LEVEL` / `LOG_FORMAT` | default production / info / json |

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
