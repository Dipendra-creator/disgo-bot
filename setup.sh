#!/usr/bin/env bash
#
# disgo-bot — one-shot installer for a fresh Ubuntu VM (EC2, Lightsail, any VPS).
#
# It asks for the bot's configuration first, then installs the prebuilt binary
# shipped in this branch as a systemd service, optionally provisioning a local
# PostgreSQL. Re-running it updates the config and binary in place.
#
#   sudo ./setup.sh            # install / update
#   sudo ./setup.sh uninstall  # stop + remove service, binary and config
#
# No Go toolchain or source checkout is required — the binary is self-contained
# (static, with HTML/CSS/JS assets and SQL migrations embedded). Migrations run
# automatically on first start.

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
# Prompt helpers
#   ask     VAR "Question" "default"   -> plain, default echoed
#   ask_req VAR "Question"             -> required, loops until non-empty
#   ask_sec VAR "Question" [required]  -> hidden input
#   ask_yn  VAR "Question" "yes|no"    -> sets VAR to "true"/"false"
# Existing value of VAR (e.g. loaded from an env file) becomes the default.
# ---------------------------------------------------------------------------
ask() {
  local __v=$1 q=$2 def=${3:-} cur=${!1:-} ans
  [ -n "$cur" ] && def=$cur
  read -r -p "$(printf '%s %s' "$q" "${def:+[${def}] }")" ans || true
  printf -v "$__v" '%s' "${ans:-$def}"
}
ask_req() {
  local __v=$1 q=$2
  while :; do ask "$__v" "$q"; [ -n "${!1:-}" ] && break; warn "required"; done
}
ask_sec() {
  local __v=$1 q=$2 req=${3:-} cur=${!1:-} ans
  while :; do
    read -r -s -p "$(printf '%s %s' "$q" "${cur:+[keep existing] }")" ans || true
    echo
    if [ -z "$ans" ] && [ -n "$cur" ]; then printf -v "$__v" '%s' "$cur"; break; fi
    if [ -n "$ans" ]; then printf -v "$__v" '%s' "$ans"; break; fi
    [ -z "$req" ] && { printf -v "$__v" '%s' ""; break; }
    warn "required"
  done
}
ask_yn() {
  local __v=$1 q=$2 def=${3:-no} cur=${!1:-} ans d
  case "$cur" in true) def=yes;; false) def=no;; esac
  case "$def" in yes) d=Y/n;; *) d=y/N;; esac
  read -r -p "$(printf '%s [%s] ' "$q" "$d")" ans || true
  ans=${ans:-$def}
  case "${ans,,}" in y|yes|true) printf -v "$__v" 'true';; *) printf -v "$__v" 'false';; esac
}

# ---------------------------------------------------------------------------
# Pre-flight
# ---------------------------------------------------------------------------
[ "$(id -u)" -eq 0 ] || die "run as root:  sudo $0 ${1:-}"

case "$(uname -m)" in
  x86_64|amd64)  ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) die "unsupported CPU architecture: $(uname -m) (need x86_64 or aarch64)" ;;
esac
BIN_SRC="${SCRIPT_DIR}/${APP}-linux-${ARCH}"

# ---------------------------------------------------------------------------
# uninstall
# ---------------------------------------------------------------------------
if [ "${1:-}" = "uninstall" ]; then
  say "Uninstalling ${APP}"
  systemctl stop "${APP}" 2>/dev/null || true
  systemctl disable "${APP}" 2>/dev/null || true
  rm -f "$SERVICE" "$BIN_DEST"
  systemctl daemon-reload
  warn "left in place: ${ENV_DIR} (secrets) and ${DATA_DIR}, and any local PostgreSQL."
  warn "remove manually if wanted:  rm -rf ${ENV_DIR} ${DATA_DIR}"
  ok "service removed"
  exit 0
fi

[ -f "$BIN_SRC" ] || die "binary not found: ${BIN_SRC}
Run this from a checkout of the build branch (binary must sit beside setup.sh)."

# ---------------------------------------------------------------------------
# Load existing config as prompt defaults (re-run / update path)
# ---------------------------------------------------------------------------
if [ -f "$ENV_FILE" ]; then
  say "Existing config found at ${ENV_FILE} — values become defaults"
  # shellcheck disable=SC1090
  set -a; . "$ENV_FILE"; set +a
fi

# Best-effort public address for the OAuth redirect default.
detect_host() {
  local ip
  ip=$(curl -fsS --max-time 2 http://169.254.169.254/latest/meta-data/public-ipv4 2>/dev/null) || true
  [ -z "$ip" ] && ip=$(curl -fsS --max-time 2 https://api.ipify.org 2>/dev/null) || true
  [ -z "$ip" ] && ip=$(hostname -I 2>/dev/null | awk '{print $1}') || true
  printf '%s' "$ip"
}

echo
say "${B}disgo-bot configuration${RST}  (press Enter to accept the [default])"
echo "${DIM}   Secrets are stored in ${ENV_FILE} with 0600 perms, never printed.${RST}"
echo

# --- Discord -------------------------------------------------------------
say "Discord credentials"
ask_req DISCORD_APP_ID      "  Application (client) ID"
ask_sec DISCORD_TOKEN       "  Bot token"            required
ask     DISCORD_DEV_GUILD_ID "  Dev guild ID (blank = register commands globally)"
ask_yn  DISCORD_PRIVILEGED_INTENTS "  Enable privileged intents (members + message content)?" no

# --- Database ------------------------------------------------------------
echo
say "Database"
DB_MODE=""
if [ -n "${DATABASE_URL:-}" ]; then
  echo "  current DATABASE_URL is set."
fi
echo "  1) Install & configure PostgreSQL on this VM (recommended for a single box)"
echo "  2) Use an existing PostgreSQL (RDS / managed / remote) — paste a DATABASE_URL"
read -r -p "  choice [1]: " DB_MODE || true
DB_MODE=${DB_MODE:-1}

if [ "$DB_MODE" = "2" ]; then
  ask_req DATABASE_URL "  DATABASE_URL (postgres://user:pass@host:5432/db?sslmode=require)"
  INSTALL_PG=false
else
  INSTALL_PG=true
  ask PG_DB   "  Database name" disgo
  ask PG_USER "  Database user" disgo
  # Reuse a previously generated password if we can recover it from DATABASE_URL.
  PG_PASS=""
  if [ -n "${DATABASE_URL:-}" ]; then
    PG_PASS=$(printf '%s' "$DATABASE_URL" | sed -nE 's#^postgres://[^:]+:([^@]+)@.*#\1#p') || true
  fi
  [ -z "$PG_PASS" ] && PG_PASS=$(openssl rand -hex 16)
  DATABASE_URL="postgres://${PG_USER}:${PG_PASS}@127.0.0.1:5432/${PG_DB}?sslmode=disable"
fi

# --- Web dashboard -------------------------------------------------------
echo
say "Web dashboard"
ask_yn WEB_ENABLED "  Enable the dashboard?" yes
if [ "$WEB_ENABLED" = "true" ]; then
  ask     WEB_ADDR    "  Listen address" ":8081"
  ask_sec DISCORD_CLIENT_SECRET "  Discord OAuth2 client secret" required
  HOST_DEF=$(detect_host); PORT=${WEB_ADDR#:}
  ask     WEB_PUBLIC_URL "  Public base URL (OAuth redirect built from this)" "http://${HOST_DEF}:${PORT}"
  case "$WEB_PUBLIC_URL" in https://*) ask_yn WEB_COOKIE_SECURE "  Mark session cookie Secure (HTTPS-only)?" yes;;
                            *)         WEB_COOKIE_SECURE=false;; esac
fi

# --- AI module (optional) ------------------------------------------------
echo
say "AI assistant module (optional)"
ask_yn AI_ENABLED "  Enable the Anthropic-backed AI module?" no
if [ "$AI_ENABLED" = "true" ]; then
  ask_sec ANTHROPIC_API_KEY "  Anthropic API key" required
  ask     AI_MODEL "  Model" "claude-opus-4-8"
fi

# --- Runtime -------------------------------------------------------------
echo
say "Runtime"
ask DISGO_ENV  "  Environment (development|production)" production
ask LOG_LEVEL  "  Log level (debug|info|warn|error)" info
ask LOG_FORMAT "  Log format (json|console)" json

# ---------------------------------------------------------------------------
# Install OS packages
# ---------------------------------------------------------------------------
echo
say "Installing system packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
PKGS="ca-certificates curl"
$INSTALL_PG && PKGS="$PKGS postgresql"
apt-get install -y -qq $PKGS >/dev/null
ok "packages installed"

# ---------------------------------------------------------------------------
# Provision local PostgreSQL
# ---------------------------------------------------------------------------
if $INSTALL_PG; then
  say "Configuring PostgreSQL ( db=${PG_DB} user=${PG_USER} )"
  systemctl enable --now postgresql >/dev/null 2>&1 || true
  # Idempotent role + database creation.
  sudo -u postgres psql -v ON_ERROR_STOP=1 <<SQL >/dev/null
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${PG_USER}') THEN
    CREATE ROLE "${PG_USER}" LOGIN PASSWORD '${PG_PASS}';
  ELSE
    ALTER ROLE "${PG_USER}" WITH LOGIN PASSWORD '${PG_PASS}';
  END IF;
END
\$\$;
SQL
  if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='${PG_DB}'" | grep -q 1; then
    sudo -u postgres createdb -O "${PG_USER}" "${PG_DB}"
  fi
  ok "database ready (local, sslmode=disable, loopback only)"
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
# Env file (0600, root:root)
# ---------------------------------------------------------------------------
say "Writing ${ENV_FILE}"
install -d -m 0750 "$ENV_DIR"
umask 077
{
  echo "# disgo-bot environment — generated by setup.sh. Contains secrets."
  echo "DISGO_ENV=${DISGO_ENV}"
  echo "LOG_LEVEL=${LOG_LEVEL}"
  echo "LOG_FORMAT=${LOG_FORMAT}"
  echo
  echo "DISCORD_APP_ID=${DISCORD_APP_ID}"
  echo "DISCORD_TOKEN=${DISCORD_TOKEN}"
  echo "DISCORD_DEV_GUILD_ID=${DISCORD_DEV_GUILD_ID}"
  echo "DISCORD_PRIVILEGED_INTENTS=${DISCORD_PRIVILEGED_INTENTS}"
  echo
  echo "DATABASE_URL=${DATABASE_URL}"
  echo
  echo "WEB_ENABLED=${WEB_ENABLED}"
  if [ "$WEB_ENABLED" = "true" ]; then
    echo "WEB_ADDR=${WEB_ADDR}"
    echo "WEB_PUBLIC_URL=${WEB_PUBLIC_URL}"
    echo "WEB_COOKIE_SECURE=${WEB_COOKIE_SECURE}"
    echo "DISCORD_CLIENT_SECRET=${DISCORD_CLIENT_SECRET}"
  fi
  echo
  echo "AI_ENABLED=${AI_ENABLED}"
  if [ "$AI_ENABLED" = "true" ]; then
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

# Hardening
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
# Start
# ---------------------------------------------------------------------------
say "Starting service"
systemctl daemon-reload
systemctl enable "${APP}" >/dev/null 2>&1 || true
systemctl restart "${APP}"
sleep 3

echo
if systemctl is-active --quiet "${APP}"; then
  ok "${B}${APP} is running${RST}"
else
  warn "service did not stay up — recent logs:"
  journalctl -u "${APP}" -n 30 --no-pager || true
  die "startup failed (see logs above). Fix config and re-run:  sudo $0"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
PORT=${WEB_ADDR#:}
cat <<DONE

${GRN}${B}Done.${RST}

  Service     systemctl status ${APP}
  Logs        journalctl -u ${APP} -f
  Restart     systemctl restart ${APP}
  Config      ${ENV_FILE}   (edit, then: systemctl restart ${APP})
  Update      re-run this script after pulling a new build

  Ports (open these in your security group / firewall as needed)
DONE
[ "$WEB_ENABLED" = "true" ] && echo "    dashboard ${WEB_ADDR}  (${WEB_PUBLIC_URL})"
cat <<DONE2
    health    :8080/healthz       metrics   :9090/metrics

DONE2
if [ "$WEB_ENABLED" = "true" ]; then
  echo "  ${YLW}Discord portal:${RST} add this OAuth2 redirect URI, or login will fail:"
  echo "      ${WEB_PUBLIC_URL%/}/auth/callback"
  echo
fi
