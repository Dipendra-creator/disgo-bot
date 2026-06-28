#!/usr/bin/env bash
#
# disgo-bot — one-shot installer for a fresh Ubuntu VM (EC2, Lightsail, any VPS).
#
# Preferred (no prompts):
#   cp .env.example .env && nano .env && sudo ./setup.sh
# The installer reads ./.env (or --env PATH), installs the prebuilt binary as a
# systemd service, and — unless you point it at an existing database — installs
# and provisions a local PostgreSQL. With no env file it falls back to an
# interactive wizard.
#
#   sudo ./setup.sh                 # use ./.env, else ask interactively
#   sudo ./setup.sh --env prod.env  # use a specific env file
#   sudo ./setup.sh uninstall       # stop + remove service, binary, unit
#
# No Go toolchain or source checkout is required — the binary is self-contained
# (static, dashboard assets + SQL migrations embedded). Migrations run on start.

set -euo pipefail

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
APP=disgo-bot
SERVICE=/etc/systemd/system/${APP}.service
ENV_DIR=/etc/${APP}
ENV_FILE=${ENV_DIR}/${APP}.env
BIN_DEST=/usr/local/bin/${APP}
DATA_DIR=/var/lib/${APP}
RUN_USER=disgo
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ---------------------------------------------------------------------------
# Pretty output
# ---------------------------------------------------------------------------
if [ -t 1 ]; then
  B=$'\033[1m'; DIM=$'\033[2m'; GRN=$'\033[32m'; YLW=$'\033[33m'; RED=$'\033[31m'; RST=$'\033[0m'
else
  B=""; DIM=""; GRN=""; YLW=""; RED=""; RST=""
fi
say()  { printf '%s\n' "${B}==>${RST} $*"; }
ok()   { printf '%s\n' "${GRN}  ok${RST} $*"; }
warn() { printf '%s\n' "${YLW}  ! ${RST} $*"; }
die()  { printf '%s\n' "${RED}error:${RST} $*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
boolnorm() { case "${!1:-}" in [Tt]rue|1|[Yy]|[Yy]es|[Oo]n) printf -v "$1" true;; *) printf -v "$1" false;; esac; }
resolve()  { [ -n "${!1:-}" ] || printf -v "$1" '%s' "${2:-}"; }   # resolve VAR default
gen_pass() { openssl rand -hex 16 2>/dev/null || tr -dc 'a-f0-9' </dev/urandom | head -c32; }

# Parse a KEY=value env file robustly: strips comments, blank lines, a Windows
# CR, leading 'export ', and a single layer of surrounding quotes. Exports each
# key into the current shell. This is what makes a pasted/Windows-edited token
# work — a stray CR on DISCORD_TOKEN is exactly what Discord rejects as 4004.
load_env() {
  local line key val
  while IFS= read -r line || [ -n "$line" ]; do
    line=${line%$'\r'}
    line=${line#"${line%%[![:space:]]*}"}      # ltrim
    case "$line" in ''|\#*) continue ;; esac
    line=${line#export }
    case "$line" in *=*) ;; *) continue ;; esac
    key=${line%%=*}; val=${line#*=}
    key=${key//[[:space:]]/}
    val=${val%$'\r'}
    val=${val#"${val%%[![:space:]]*}"}         # ltrim value
    val=${val%"${val##*[![:space:]]}"}         # rtrim value
    case "$val" in
      \"*\") val=${val#\"}; val=${val%\"} ;;
      \'*\') val=${val#\'}; val=${val%\'} ;;
    esac
    printf -v "$key" '%s' "$val"
    export "$key"
  done < "$1"
}

detect_host() {
  local ip
  ip=$(curl -fsS --max-time 2 http://169.254.169.254/latest/meta-data/public-ipv4 2>/dev/null) || true
  [ -z "$ip" ] && ip=$(curl -fsS --max-time 2 https://api.ipify.org 2>/dev/null) || true
  [ -z "$ip" ] && ip=$(hostname -I 2>/dev/null | awk '{print $1}') || true
  printf '%s' "$ip"
}

# ---------------------------------------------------------------------------
# Pre-flight + args
# ---------------------------------------------------------------------------
[ "$(id -u)" -eq 0 ] || die "run as root:  sudo $0 $*"

CLI_ENV=""
case "${1:-}" in
  uninstall) MODE=uninstall ;;
  --env)     MODE=install; CLI_ENV=${2:-} ; [ -n "$CLI_ENV" ] || die "--env needs a path" ;;
  "")        MODE=install ;;
  *)         die "unknown argument: $1  (use: install | --env PATH | uninstall)" ;;
esac

case "$(uname -m)" in
  x86_64|amd64)  ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) die "unsupported CPU architecture: $(uname -m) (need x86_64 or aarch64)" ;;
esac
BIN_SRC="${SCRIPT_DIR}/${APP}-linux-${ARCH}"

# ---------------------------------------------------------------------------
# uninstall
# ---------------------------------------------------------------------------
if [ "$MODE" = uninstall ]; then
  say "Uninstalling ${APP}"
  systemctl stop "${APP}" 2>/dev/null || true
  systemctl disable "${APP}" 2>/dev/null || true
  rm -f "$SERVICE" "$BIN_DEST"
  systemctl daemon-reload
  warn "left in place: ${ENV_DIR} (secrets), ${DATA_DIR}, and any local PostgreSQL."
  warn "remove manually if wanted:  rm -rf ${ENV_DIR} ${DATA_DIR}"
  ok "service removed"
  exit 0
fi

[ -f "$BIN_SRC" ] || die "binary not found: ${BIN_SRC}
Run this from a checkout of the build branch (binary must sit beside setup.sh)."

# ---------------------------------------------------------------------------
# Load configuration
#   priority: --env PATH  >  ./.env  >  existing /etc/disgo-bot/disgo-bot.env
#   if any is found  -> non-interactive (env-driven)
#   if none is found -> interactive wizard
# ---------------------------------------------------------------------------
ENV_SRC=""
for cand in "$CLI_ENV" "${SCRIPT_DIR}/.env" "./.env" "$ENV_FILE"; do
  if [ -n "$cand" ] && [ -f "$cand" ]; then ENV_SRC="$cand"; break; fi
done

if [ -n "$ENV_SRC" ]; then
  say "Reading configuration from ${B}${ENV_SRC}${RST}"
  load_env "$ENV_SRC"
  INTERACTIVE=false
else
  INTERACTIVE=true
fi

if [ "$INTERACTIVE" = true ]; then
  # ---- interactive wizard (no env file present) -------------------------
  ask() { local __v=$1 q=$2 def=${3:-} cur=${!1:-} a; [ -n "$cur" ] && def=$cur
          read -r -p "$(printf '%s %s' "$q" "${def:+[${def}] }")" a || true; printf -v "$__v" '%s' "${a:-$def}"; }
  ask_sec() { local __v=$1 q=$2 req=${3:-} cur=${!1:-} a
    while :; do read -r -s -p "$(printf '%s %s' "$q" "${cur:+[keep] }")" a || true; echo
      if [ -z "$a" ] && [ -n "$cur" ]; then printf -v "$__v" '%s' "$cur"; break; fi
      if [ -n "$a" ]; then printf -v "$__v" '%s' "$a"; break; fi
      [ -z "$req" ] && { printf -v "$__v" '%s' ""; break; }; warn "required"; done; }
  ask_yn() { local __v=$1 q=$2 def=${3:-no} cur=${!1:-} a d; case "$cur" in true)def=yes;;false)def=no;;esac
    case "$def" in yes)d=Y/n;;*)d=y/N;;esac; read -r -p "$(printf '%s [%s] ' "$q" "$d")" a || true; a=${a:-$def}
    case "${a,,}" in y|yes|true) printf -v "$__v" true;; *) printf -v "$__v" false;; esac; }

  echo; say "${B}disgo-bot configuration${RST}  (Enter accepts the [default])"
  echo "${DIM}   Tip: skip these prompts next time — cp .env.example .env, fill it, re-run.${RST}"; echo
  say "Discord credentials"
  ask     DISCORD_APP_ID "  Application (client) ID"
  ask_sec DISCORD_TOKEN  "  Bot token" required
  ask     DISCORD_DEV_GUILD_ID "  Dev guild ID (blank = global commands)"
  ask_yn  DISCORD_PRIVILEGED_INTENTS "  Enable privileged intents?" no
  echo; say "Database"
  echo "  1) Install & configure PostgreSQL on this VM"
  echo "  2) Use an existing PostgreSQL — paste a DATABASE_URL"
  read -r -p "  choice [1]: " c || true
  if [ "${c:-1}" = 2 ]; then ask DATABASE_URL "  DATABASE_URL"; PG_INSTALL=false
  else PG_INSTALL=true; ask PG_DB "  Database name" disgo; ask PG_USER "  Database user" disgo; fi
  echo; say "Web dashboard"
  ask_yn WEB_ENABLED "  Enable the dashboard?" yes
  if [ "$WEB_ENABLED" = true ]; then
    ask     WEB_ADDR "  Listen address" ":8081"
    ask_sec DISCORD_CLIENT_SECRET "  Discord OAuth2 client secret" required
    ask     WEB_PUBLIC_URL "  Public base URL" "http://$(detect_host):${WEB_ADDR#:}"
    case "$WEB_PUBLIC_URL" in https://*) ask_yn WEB_COOKIE_SECURE "  Secure cookie?" yes;; *) WEB_COOKIE_SECURE=false;; esac
  fi
  echo; say "AI assistant module (optional)"
  ask_yn AI_ENABLED "  Enable the AI module?" no
  if [ "$AI_ENABLED" = true ]; then ask_sec ANTHROPIC_API_KEY "  Anthropic API key" required; ask AI_MODEL "  Model" "claude-opus-4-8"; fi
  echo; say "Runtime"
  ask DISGO_ENV  "  Environment (development|production)" production
  ask LOG_LEVEL  "  Log level" info
  ask LOG_FORMAT "  Log format (json|console)" json
fi

# ---------------------------------------------------------------------------
# Normalize + apply defaults (covers both modes)
# ---------------------------------------------------------------------------
resolve DISGO_ENV production
resolve LOG_LEVEL info
resolve LOG_FORMAT json
resolve WEB_ADDR :8081
resolve AI_MODEL claude-opus-4-8
resolve PG_DB disgo
resolve PG_USER disgo
resolve WEB_ENABLED true
resolve PG_INSTALL true
resolve DISCORD_PRIVILEGED_INTENTS false
resolve AI_ENABLED false
resolve WEB_COOKIE_SECURE false
for b in DISCORD_PRIVILEGED_INTENTS WEB_ENABLED WEB_COOKIE_SECURE AI_ENABLED PG_INSTALL; do boolnorm "$b"; done

# Database resolution: explicit DATABASE_URL wins; otherwise provision local PG.
INSTALL_PG=false
if [ -n "${DATABASE_URL:-}" ]; then
  INSTALL_PG=false
elif [ "${PG_INSTALL:-true}" = true ]; then
  INSTALL_PG=true
  [ -n "${PG_PASSWORD:-}" ] || PG_PASSWORD=$(gen_pass)
  DATABASE_URL="postgres://${PG_USER}:${PG_PASSWORD}@127.0.0.1:5432/${PG_DB}?sslmode=disable"
fi

# Web public URL: auto-detect if enabled but unset.
if [ "$WEB_ENABLED" = true ] && [ -z "${WEB_PUBLIC_URL:-}" ]; then
  WEB_PUBLIC_URL="http://$(detect_host):${WEB_ADDR#:}"
fi

# ---------------------------------------------------------------------------
# Validate required values (fail fast, list everything missing at once)
# ---------------------------------------------------------------------------
MISSING=()
[ -n "${DISCORD_APP_ID:-}" ] || MISSING+=("DISCORD_APP_ID")
[ -n "${DISCORD_TOKEN:-}" ]  || MISSING+=("DISCORD_TOKEN")
[ -n "${DATABASE_URL:-}" ]   || MISSING+=("DATABASE_URL (or PG_INSTALL=true)")
[ "$WEB_ENABLED" = true ] && [ -z "${DISCORD_CLIENT_SECRET:-}" ] && MISSING+=("DISCORD_CLIENT_SECRET (dashboard is on)")
[ "$AI_ENABLED" = true ]  && [ -z "${ANTHROPIC_API_KEY:-}" ]     && MISSING+=("ANTHROPIC_API_KEY (AI is on)")
if [ "${#MISSING[@]}" -gt 0 ]; then
  printf '%s\n' "${RED}error:${RST} missing required configuration:" >&2
  for m in "${MISSING[@]}"; do printf '   - %s\n' "$m" >&2; done
  [ "$INTERACTIVE" = false ] && printf '%s\n' "Edit ${ENV_SRC} and re-run." >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Install OS packages (idempotent; fresh-VM safe)
# ---------------------------------------------------------------------------
echo; say "Installing system packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
PKGS="ca-certificates curl openssl"
$INSTALL_PG && PKGS="$PKGS postgresql postgresql-contrib"
apt-get install -y -qq $PKGS >/dev/null
ok "packages installed"

# ---------------------------------------------------------------------------
# Provision local PostgreSQL
# ---------------------------------------------------------------------------
if $INSTALL_PG; then
  say "Configuring PostgreSQL ( db=${PG_DB} user=${PG_USER} )"
  systemctl enable --now postgresql >/dev/null 2>&1 || true
  # Wait for the cluster to accept connections (fresh installs init lazily).
  ready=false
  for _ in $(seq 1 30); do
    if sudo -u postgres psql -tAc 'SELECT 1' >/dev/null 2>&1; then ready=true; break; fi
    sleep 1
  done
  $ready || die "PostgreSQL did not become ready — check: systemctl status postgresql"
  sudo -u postgres psql -v ON_ERROR_STOP=1 \
      -v usr="${PG_USER}" -v pw="${PG_PASSWORD}" <<'SQL' >/dev/null
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'usr', :'pw')
  WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'usr') \gexec
SELECT format('ALTER ROLE %I LOGIN PASSWORD %L', :'usr', :'pw') \gexec
SQL
  if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='${PG_DB}'" | grep -q 1; then
    sudo -u postgres createdb -O "${PG_USER}" "${PG_DB}"
  fi
  ok "database ready (local, loopback only)"
fi

# ---------------------------------------------------------------------------
# Service user, dirs, binary
# ---------------------------------------------------------------------------
say "Installing binary and service user"
id -u "$RUN_USER" >/dev/null 2>&1 || useradd --system --no-create-home --shell /usr/sbin/nologin "$RUN_USER"
install -d -o "$RUN_USER" -g "$RUN_USER" -m 750 "$DATA_DIR"
systemctl stop "${APP}" 2>/dev/null || true
install -m 0755 "$BIN_SRC" "$BIN_DEST"
ok "binary -> ${BIN_DEST} (${ARCH})"

# ---------------------------------------------------------------------------
# Env file consumed by systemd (0600, root)
# ---------------------------------------------------------------------------
say "Writing ${ENV_FILE}"
install -d -m 0750 "$ENV_DIR"
umask 077
{
  echo "# disgo-bot environment — generated by setup.sh. Contains secrets."
  echo "DISGO_ENV=${DISGO_ENV}"
  echo "LOG_LEVEL=${LOG_LEVEL}"
  echo "LOG_FORMAT=${LOG_FORMAT}"
  echo "DISCORD_APP_ID=${DISCORD_APP_ID}"
  echo "DISCORD_TOKEN=${DISCORD_TOKEN}"
  echo "DISCORD_DEV_GUILD_ID=${DISCORD_DEV_GUILD_ID:-}"
  echo "DISCORD_PRIVILEGED_INTENTS=${DISCORD_PRIVILEGED_INTENTS}"
  echo "DATABASE_URL=${DATABASE_URL}"
  echo "WEB_ENABLED=${WEB_ENABLED}"
  if [ "$WEB_ENABLED" = true ]; then
    echo "WEB_ADDR=${WEB_ADDR}"
    echo "WEB_PUBLIC_URL=${WEB_PUBLIC_URL}"
    echo "WEB_COOKIE_SECURE=${WEB_COOKIE_SECURE}"
    echo "DISCORD_CLIENT_SECRET=${DISCORD_CLIENT_SECRET}"
  fi
  echo "AI_ENABLED=${AI_ENABLED}"
  if [ "$AI_ENABLED" = true ]; then
    echo "ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}"
    echo "AI_MODEL=${AI_MODEL}"
  fi
} > "$ENV_FILE"
chmod 600 "$ENV_FILE"; chown root:root "$ENV_FILE"
ok "config written (0600)"

# ---------------------------------------------------------------------------
# systemd unit
# ---------------------------------------------------------------------------
say "Writing systemd unit"
cat > "$SERVICE" <<UNIT
[Unit]
Description=disgo-bot Discord bot
Documentation=https://github.com/Dipendra-creator/disgo-bot
After=network-online.target postgresql.service
Wants=network-online.target

[Service]
Type=simple
User=${RUN_USER}
Group=${RUN_USER}
WorkingDirectory=${DATA_DIR}
EnvironmentFile=${ENV_FILE}
ExecStart=${BIN_DEST}
Restart=on-failure
RestartSec=5
TimeoutStopSec=20
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ProtectKernelTunables=true
ProtectControlGroups=true
RestrictSUIDSGID=true
ReadWritePaths=${DATA_DIR}

[Install]
WantedBy=multi-user.target
UNIT
ok "unit -> ${SERVICE}"

# ---------------------------------------------------------------------------
# Start + verify
# ---------------------------------------------------------------------------
say "Starting service"
systemctl daemon-reload
systemctl enable "${APP}" >/dev/null 2>&1 || true
systemctl restart "${APP}"
sleep 4

echo
if systemctl is-active --quiet "${APP}"; then
  ok "${B}${APP} is running${RST}"
else
  warn "service did not stay up — recent logs:"
  journalctl -u "${APP}" -n 20 --no-pager || true
  echo
  if journalctl -u "${APP}" -n 40 --no-pager 2>/dev/null | grep -q '4004'; then
    die "Discord rejected the bot token (close 4004 Authentication failed).
DISCORD_TOKEN is wrong, expired, or from a different application than DISCORD_APP_ID.
Reset it in the Developer Portal (Bot → Reset Token), update ${ENV_SRC:-your env}, re-run."
  fi
  if journalctl -u "${APP}" -n 40 --no-pager 2>/dev/null | grep -q '4014'; then
    die "Discord disallowed the requested intents (close 4014).
Enable the privileged intents in the portal (Bot → Privileged Gateway Intents),
or set DISCORD_PRIVILEGED_INTENTS=false, then re-run."
  fi
  die "startup failed (logs above). Fix config and re-run:  sudo $0"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
cat <<DONE

${GRN}${B}Done.${RST}

  Status   systemctl status ${APP}
  Logs     journalctl -u ${APP} -f
  Config   ${ENV_FILE}   (edit, then: systemctl restart ${APP})
  Update   pull a new build, re-run this script

  Ports (open in your EC2 security group as needed)
DONE
[ "$WEB_ENABLED" = true ] && echo "    dashboard ${WEB_ADDR}   ${WEB_PUBLIC_URL}"
cat <<DONE2
    health    :8080/healthz       metrics   :9090/metrics

DONE2
if [ "$WEB_ENABLED" = true ]; then
  echo "  ${YLW}Discord portal:${RST} add this OAuth2 redirect URI or login fails:"
  echo "      ${WEB_PUBLIC_URL%/}/auth/callback"
  echo
fi
