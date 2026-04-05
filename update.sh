#!/usr/bin/env bash
set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
fail()  { echo -e "${RED}[FAIL]${NC}  $*"; exit 1; }

cd "$(dirname "$0")"

echo -e "${CYAN}"
echo "  BOTLOG — Update"
echo -e "${NC}"

# ��─ Detect compose command ─────────���─────────────────────────────
if docker compose version &>/dev/null; then
    COMPOSE="docker compose"
elif command -v docker-compose &>/dev/null; then
    COMPOSE="docker-compose"
else
    fail "Docker Compose not found."
fi

# ── 1. Git pull ─────────────────────────────────────────────────
SELF_HASH=$(md5sum "$0" 2>/dev/null || md5 -q "$0" 2>/dev/null)

info "Pulling latest code..."
if ! git pull --ff-only; then
    fail "git pull failed. Resolve conflicts manually and re-run."
fi
ok "Code updated"

NEW_HASH=$(md5sum "$0" 2>/dev/null || md5 -q "$0" 2>/dev/null)
if [ "$SELF_HASH" != "$NEW_HASH" ]; then
    warn "update.sh itself was updated. Please re-run: ./update.sh"
    exit 0
fi

# ── 2. Backup ClickHouse data ──────────────────────────────────
BACKUP_DIR="$(pwd)/backups"
mkdir -p "$BACKUP_DIR"
BACKUP_TS=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/botlog_${BACKUP_TS}.sql.gz"

if docker inspect -f '{{.State.Running}}' botlog-clickhouse 2>/dev/null | grep -q true; then
    info "Backing up ClickHouse data..."
    docker exec botlog-clickhouse clickhouse-client --query \
        "SELECT * FROM botlog.hits FORMAT Native" 2>/dev/null | gzip > "$BACKUP_FILE"

    if [ -s "$BACKUP_FILE" ]; then
        ok "Backup saved: $BACKUP_FILE ($(du -h "$BACKUP_FILE" | cut -f1))"
    else
        rm -f "$BACKUP_FILE"
        warn "Backup is empty (no data yet?) — continuing anyway"
    fi

    # Keep only last 5 backups
    ls -1t "$BACKUP_DIR"/botlog_*.sql.gz 2>/dev/null | tail -n +6 | xargs -r rm -f
else
    warn "ClickHouse not running — skipping backup"
fi

# ── 3. Update GeoLite2-City.mmdb ───────────────────────────────
DATA_DIR="$(pwd)/data"
MMDB_PATH="${DATA_DIR}/GeoLite2-City.mmdb"
GEOLITE2_URL="https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb"

mkdir -p "$DATA_DIR"

info "Downloading latest GeoLite2-City.mmdb..."
MMDB_TMP="${MMDB_PATH}.tmp"

if command -v curl &>/dev/null; then
    curl -fSL --progress-bar -o "$MMDB_TMP" "$GEOLITE2_URL"
elif command -v wget &>/dev/null; then
    wget -q --show-progress -O "$MMDB_TMP" "$GEOLITE2_URL"
else
    fail "Neither curl nor wget found."
fi

if [ ! -s "$MMDB_TMP" ]; then
    rm -f "$MMDB_TMP"
    fail "Download failed or file is empty."
fi

mv -f "$MMDB_TMP" "$MMDB_PATH"
ok "GeoLite2-City.mmdb updated ($(du -h "$MMDB_PATH" | cut -f1))"

# ── 4. Rebuild and restart (backend + frontend only, never touch ClickHouse) ──
info "Rebuilding backend and frontend (ClickHouse untouched)..."
$COMPOSE up -d --build --no-deps backend frontend 2>&1 | while IFS= read -r line; do
    echo -e "  ${line}"
done

# Ensure ClickHouse is running (start if stopped, but never recreate)
if ! docker inspect -f '{{.State.Running}}' botlog-clickhouse 2>/dev/null | grep -q true; then
    info "Starting ClickHouse..."
    $COMPOSE up -d clickhouse 2>&1 | while IFS= read -r line; do
        echo -e "  ${line}"
    done
fi

echo ""
ok "Update complete. Dashboard: http://localhost"
echo -e "  ClickHouse was NOT rebuilt — your data is safe."
echo -e "  To view logs: ${YELLOW}$COMPOSE logs -f${NC}"
