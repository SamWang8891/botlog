#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
fail()  { echo -e "${RED}[FAIL]${NC}  $*"; exit 1; }

cd "$(dirname "$0")"

BACKUP_DIR="$(pwd)/backups"
mkdir -p "$BACKUP_DIR"
BACKUP_TS=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/botlog_${BACKUP_TS}.sql.gz"

if ! docker inspect -f '{{.State.Running}}' botlog-clickhouse 2>/dev/null | grep -q true; then
    fail "ClickHouse container is not running."
fi

info "Backing up ClickHouse data..."
docker exec botlog-clickhouse clickhouse-client --query \
    "SELECT * FROM botlog.hits FORMAT Native" 2>/dev/null | gzip > "$BACKUP_FILE"

if [ ! -s "$BACKUP_FILE" ]; then
    rm -f "$BACKUP_FILE"
    fail "Backup is empty — no data in the database."
fi

ok "Backup saved: $BACKUP_FILE ($(du -h "$BACKUP_FILE" | cut -f1))"

# Show recent backups
echo ""
echo -e "  ${CYAN}Recent backups:${NC}"
ls -1t "$BACKUP_DIR"/botlog_*.sql.gz 2>/dev/null | head -5 | while read -r f; do
    echo -e "    $(du -h "$f" | cut -f1)\t$(basename "$f")"
done

echo ""
echo -e "  ${CYAN}To restore:${NC}"
echo -e "    ${YELLOW}gunzip -c $BACKUP_FILE | docker exec -i botlog-clickhouse clickhouse-client --query 'INSERT INTO botlog.hits FORMAT Native'${NC}"
