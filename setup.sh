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

echo -e "${CYAN}"
echo "  ____   ___ _____ _     ___   ____ "
echo " | __ ) / _ \_   _| |   / _ \ / ___|"
echo " |  _ \| | | || | | |  | | | | |  _ "
echo " | |_) | |_| || | | |__| |_| | |_| |"
echo " |____/ \___/ |_| |_____\___/ \____|"
echo -e "${NC}"
echo "  Real-Time Bot Traffic Monitor — Setup"
echo ""

# ── 1. Check Docker ──────────────────────────────────────────────
info "Checking for Docker..."
if ! command -v docker &>/dev/null; then
    fail "Docker is not installed. Please install Docker first: https://docs.docker.com/engine/install/"
fi
ok "Docker found: $(docker --version)"

# ── 2. Check Docker Compose ──────────────────────────────────────
info "Checking for Docker Compose..."
if docker compose version &>/dev/null; then
    COMPOSE="docker compose"
elif command -v docker-compose &>/dev/null; then
    COMPOSE="docker-compose"
else
    fail "Docker Compose is not installed. Please install it: https://docs.docker.com/compose/install/"
fi
ok "Docker Compose found: $($COMPOSE version)"

# ── 3. Check Docker daemon is running ────────────────────────────
info "Checking Docker daemon..."
if ! docker info &>/dev/null; then
    fail "Docker daemon is not running. Please start Docker and try again."
fi
ok "Docker daemon is running"

# ── 4. Download GeoLite2-City.mmdb ───────────────────────────────
DATA_DIR="$(cd "$(dirname "$0")" && pwd)/data"
MMDB_PATH="${DATA_DIR}/GeoLite2-City.mmdb"
GEOLITE2_URL="https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb"

mkdir -p "$DATA_DIR"

if [ -f "$MMDB_PATH" ]; then
    ok "GeoLite2-City.mmdb already exists, skipping download"
else
    info "Downloading GeoLite2-City.mmdb..."

    if command -v curl &>/dev/null; then
        curl -fSL --progress-bar -o "$MMDB_PATH" "$GEOLITE2_URL"
    elif command -v wget &>/dev/null; then
        wget -q --show-progress -O "$MMDB_PATH" "$GEOLITE2_URL"
    else
        fail "Neither curl nor wget found. Please install one and try again."
    fi

    if [ ! -s "$MMDB_PATH" ]; then
        rm -f "$MMDB_PATH"
        fail "Download failed or file is empty. You can manually download GeoLite2-City.mmdb and place it in ${DATA_DIR}/"
    fi

    ok "GeoLite2-City.mmdb downloaded ($(du -h "$MMDB_PATH" | cut -f1))"
fi

# ── 5. Build and deploy ─────────────────────────────────────────
cd "$(dirname "$0")"

info "Building and starting services..."
$COMPOSE up -d --build 2>&1 | while IFS= read -r line; do
    echo -e "  ${line}"
done

# ── 6. Wait for services to be healthy ──────────────────────────
info "Waiting for services to start..."
sleep 5

BACKEND_OK=false
FRONTEND_OK=false

for i in $(seq 1 15); do
    if ! $BACKEND_OK && curl -sf http://localhost:8081/api/filters &>/dev/null; then
        BACKEND_OK=true
    fi
    if ! $FRONTEND_OK && curl -sf http://localhost:3000 &>/dev/null; then
        FRONTEND_OK=true
    fi
    if $BACKEND_OK && $FRONTEND_OK; then
        break
    fi
    sleep 2
done

echo ""
echo -e "${GREEN}════════════════════════════════════════════════════════════${NC}"
echo ""

if $FRONTEND_OK; then
    ok "Dashboard is live"
else
    warn "Dashboard may still be starting up"
fi

if $BACKEND_OK; then
    ok "Backend API is live"
else
    warn "Backend may still be starting up"
fi

echo ""
echo -e "  ${CYAN}Dashboard:${NC}   http://localhost:${YELLOW}3000${NC}"
echo -e "  ${CYAN}Trap port:${NC}   http://localhost:${YELLOW}8080${NC}  (point your DNS here)"
echo -e "  ${CYAN}API port:${NC}    http://localhost:${YELLOW}8081${NC}"
echo -e "  ${CYAN}ClickHouse:${NC}  http://localhost:${YELLOW}8123${NC}"
echo ""
echo -e "  To view logs:    ${YELLOW}$COMPOSE logs -f${NC}"
echo -e "  To stop:         ${YELLOW}$COMPOSE down${NC}"
echo ""
echo -e "${GREEN}════════════════════════════════════════════════════════════${NC}"
