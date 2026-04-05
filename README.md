# BOTLOG - Real-Time Bot Traffic Monitor

A standalone honeypot that logs and visualizes bot/scanner traffic in real-time.

## Architecture

- **Go backend** — dual HTTP server (trap on `:8080`, API on `:8081`)
- **ClickHouse** — columnar analytics DB for billions of rows
- **React + TypeScript** — responsive dashboard with dark/tech theme
- **MaxMind GeoLite2** — IP to city/country resolution

## Quick Start

```bash
git clone https://github.com/SamWang8891/whats-the-bot-doing.git
cd whats-the-bot-doing
./setup.sh
```

That's it. The setup script will:
1. Verify Docker and Docker Compose are installed and running
2. Download the latest GeoLite2-City.mmdb automatically
3. Build and deploy all services
4. Health-check and print the ports when ready

### Ports

| Service | Port | Description |
|---------|------|-------------|
| Dashboard | `3000` | Web UI — open in your browser |
| Trap | `8080` | Honeypot — point your DNS/firewall here |
| API | `8081` | Backend REST API + SSE stream |
| ClickHouse | `8123` | ClickHouse HTTP interface |

### Dashboard Pages

- **LIVE** — real-time scrolling feed of bot hits via SSE
- **STATS** — filterable charts (bar/line/area/pie) + CSV export

## Updating

```bash
./update.sh
```

This will:
1. Pull the latest code (`git pull --ff-only`)
2. Re-download the latest GeoLite2-City.mmdb
3. Rebuild and restart all services with zero data loss

## Configuration

Environment variables for the backend container:

| Variable | Default | Description |
|----------|---------|-------------|
| `TRAP_PORT` | `8080` | Port for the honeypot trap |
| `API_PORT` | `8081` | Port for the dashboard API |
| `CLICKHOUSE_ADDR` | `127.0.0.1:9000` | ClickHouse native address |
| `CLICKHOUSE_DB` | `botlog` | Database name |
| `GEOIP_PATH` | `/data/GeoLite2-City.mmdb` | Path to MaxMind DB |

## Development

```bash
# Backend
cd backend && go run ./cmd/main.go

# Frontend
cd frontend && npm install && npm run dev
```
